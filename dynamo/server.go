package main

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"lab3/shared"
	"log"
	"net/http"
	"net/rpc"
	"sort"
	"sync"
	"time"
)

const N_REPLICAS = 3

type Point struct {
	pos    uint32
	server string
}

type KV struct {
	mu     sync.Mutex
	vnodes int

	// Sorted ring of virtual nodes
	ring []Point

	// Per-server storage: server -> (key -> value)
	store map[string]map[string]string
}

// --- Hash helper (MD5 -> uint32) ---
func hashToUint32(s string) uint32 {
	sum := md5.Sum([]byte(s))
	return binary.BigEndian.Uint32(sum[0:4])
}

// --- Ring management ---
func (kv *KV) addNodeInternal(server string) {
	// init server storage if not present
	if _, ok := kv.store[server]; !ok {
		kv.store[server] = make(map[string]string)
	}

	// add vnode points
	for i := 0; i < kv.vnodes; i++ {
		token := fmt.Sprintf("%s#%d", server, i)
		pos := hashToUint32(token)
		kv.ring = append(kv.ring, Point{pos: pos, server: server})
	}

	// keep ring sorted
	sort.Slice(kv.ring, func(i, j int) bool {
		return kv.ring[i].pos < kv.ring[j].pos
	})
	fmt.Printf("Added %s to ring, ring now has %d vnodes\n", server, len(kv.ring))
}

func (kv *KV) deleteNodeInternal(server string) {
	// remove ring points
	newRing := kv.ring[:0]
	for _, p := range kv.ring {
		if p.server != server {
			newRing = append(newRing, p)
		}
	}
	kv.ring = newRing

	// remove storage
	delete(kv.store, server)
}

func (kv *KV) ownerForKeyInternal(key string) (string, bool) {
	if len(kv.ring) == 0 {
		return "", false
	}
	h := hashToUint32(key)

	// first vnode clockwise from h
	idx := sort.Search(len(kv.ring), func(i int) bool {
		return kv.ring[i].pos >= h
	})
	if idx == len(kv.ring) {
		idx = 0
	}
	return kv.ring[idx].server, true
}

// --- RPC methods required by assignment ---

// AddNode RPC
func (kv *KV) AddNode(args *shared.AddNodeArgs, reply *shared.AddNodeReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.addNodeInternal(args.Server)
	return nil
}

// DeleteNode RPC
func (kv *KV) DeleteNode(args *shared.DeleteNodeArgs, reply *shared.DeleteNodeReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.deleteNodeInternal(args.Server)
	return nil
}

// get(key) => return server name that stores it
func (kv *KV) GetOwner(args *shared.OwnerArgs, reply *shared.OwnerReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	owner, ok := kv.ownerForKeyInternal(args.Key)
	if !ok {
		return fmt.Errorf("no servers in ring")
	}
	reply.Server = owner
	return nil
}

// Keep your existing Get/Put RPC names, but route based on ring owner.

// Get returns the value stored on the owning server
func (kv *KV) Get(args *shared.GetArgs, reply *shared.GetReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	owner, ok := kv.ownerForKeyInternal(args.Key)
	if !ok {
		return fmt.Errorf("no servers in ring")
	}

	// If key not found, reply.Value will be empty string (same behavior as before)
	reply.Value = kv.store[owner][args.Key]
	return nil
}

// Put stores the key/value on the owning server
func (kv *KV) Put(args *shared.PutArgs, reply *shared.PutReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// write to all the replicas
	replicas := kv.preferenceListInternal(args.Key)
    fmt.Printf("Replicating key=%s to: %v\n", args.Key, replicas)

	// write to all N replicas
	for _, serverName := range replicas {
		if kv.store[serverName] == nil {
			kv.store[serverName] = make(map[string]string)
		}
		kv.store[serverName][args.Key] = args.Value
	}
	// generate a writeID to check track of a write
	writeID := fmt.Sprintf("%s-%d", args.Key, time.Now().UnixNano())

	// for the consensus for the write
	req := shared.WriteRequest{WriteID: writeID,Key: args.Key, Value: args.Value}
    var ok2 bool
    consensus.ProposeWrite(req, &ok2) 

	// after ACK_WAIT_TIME, check if enough nodes acked to declare the write a success
    go func() {
        time.Sleep(shared.ACK_WAIT_TIME * time.Millisecond)
        countAcksAndDecide(writeID)  
    }()

	return nil
}

func NewKV(vnodes int) *KV {
	return &KV{
		vnodes: vnodes,
		ring:   make([]Point, 0),
		store:  make(map[string]map[string]string),
	}
}

// checking if the majority out of the replicas wrote.
func countAcksAndDecide(writeID string) {
	var acks []shared.WriteAck
	consensus.CollectAcks(writeID, &acks)  

	ackCount := 0
	for _, ack := range acks {
		if ack.Stored && ack.WriteID == writeID {
			ackCount++
		}
	}

	var key string
	if len(acks) > 0 {
		key = acks[0].Key
	} else {
		key = "unknown"
	}

	// the write required to say succesful write (the majority of replicas)?
	W := 2
	if ackCount >= W {
        fmt.Printf("Write SUCCEEDED for key=%s got %d acks\n", key, ackCount)
    } else {
		fmt.Printf("Write FAILED for key=%s only %d acks\n", key, ackCount)
	}
}

var consensus = shared.NewQuorumTracker()

func main() {
	nodes := shared.NewMembership()
	requests := shared.NewRequests()

	kv := NewKV(4) // 4 virtual nodes per server

	// Initialize the 5 servers on the ring (names can be whatever you want)
	// commented out for now will use the connected client nodes as nodes on the ring
	// kv.addNodeInternal("Server1")
	// kv.addNodeInternal("Server2")
	// kv.addNodeInternal("Server3")
	// kv.addNodeInternal("Server4")
	// kv.addNodeInternal("Server5")

	if err := rpc.Register(nodes); err != nil {
		log.Fatal(err)
	}
	if err := rpc.Register(requests); err != nil {
		log.Fatal(err)
	}
	if err := rpc.Register(consensus); err != nil {
		log.Fatal(err)
	}
	if err := rpc.Register(kv); err != nil {
		log.Fatal(err)
	}

	rpc.HandleHTTP()

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "RPC SERVER LIVE!")
	})

	log.Println("RPC server listening on localhost:9005")
	log.Fatal(http.ListenAndServe("localhost:9005", nil))
}

func (kv *KV) preferenceListInternal(key string) []string {
	if len(kv.ring) == 0 {
		return []string{}
	}

	h := hashToUint32(key)
	idx := sort.Search(len(kv.ring), func(i int) bool {
		return kv.ring[i].pos >= h
	})
	if idx == len(kv.ring) {
		idx = 0
	}

	seen := map[string]bool{}
	list := []string{}

	for i := 0; i < len(kv.ring) && len(list) < N_REPLICAS; i++ {
		serverName := kv.ring[(idx + i) % len(kv.ring)].server
		if !seen[serverName] {
			seen[serverName] = true
			list = append(list, serverName)
		}
	}
	return list
}
 