package main

import (
	"fmt"
	"lab3/shared"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	MAX_NODES  = 8
	X_TIME     = 1
	Y_TIME     = 2
	Z_TIME_MAX = 100
	Z_TIME_MIN = 10
	FAILURE_TIMEOUT = 20 // seconds without heartbeat
)

var self_node shared.Node
var membershipMu sync.Mutex
// remeber that we already voted for that writeId
var votedWrites = make(map[string]bool)

// Send the current membership table to a neighboring node with the provided ID
func sendMessage(server *rpc.Client, id int, membership *shared.Membership) {
	mail := shared.Request{
		ID:    id,
		Table: *membership,
	}
	var ok bool
	if err := server.Call("Requests.Add", mail, &ok); err != nil {
		fmt.Println("Error: Requests.Add()", err)
	}
}

// Read incoming messages from other nodes
func readMessages(server *rpc.Client, id int, membership *shared.Membership) *shared.Membership {
	var incoming shared.Membership

	if err := server.Call("Requests.Listen", id, &incoming); err != nil {
		fmt.Println("Error: Requests.Listen()", err)
		return membership
	}
	merged := shared.CombineTables(membership, &incoming)
	return merged
}

func calcTime() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

var wg = &sync.WaitGroup{}

// -------------------- Dynamo DEMO helpers --------------------

func getOwner(server *rpc.Client, key string) string {
	args := shared.OwnerArgs{Key: key}
	reply := shared.OwnerReply{}
	err := server.Call("KV.GetOwner", &args, &reply)
	if err != nil {
		log.Fatal("KV.GetOwner error:", err)
	}
	return reply.Server
}

func deleteNode(server *rpc.Client, serverName string) {
	args := shared.DeleteNodeArgs{Server: serverName}
	reply := shared.DeleteNodeReply{}
	err := server.Call("KV.DeleteNode", &args, &reply)
	if err != nil {
		log.Fatal("KV.DeleteNode error:", err)
	}
}

func get(server *rpc.Client, key string) string {
	args := shared.GetArgs{Key: key}
	reply := shared.GetReply{}
	err := server.Call("KV.Get", &args, &reply)
	if err != nil {
		log.Fatal("KV.Get error:", err)
	}
	return reply.Value
}

func put(server *rpc.Client, key string, val string) {
	args := shared.PutArgs{Key: key, Value: val}
	reply := shared.PutReply{}
	err := server.Call("KV.Put", &args, &reply)
	if err != nil {
		log.Fatal("KV.Put error:", err)
	}
}

func dynamoDemo() {
	server, err := rpc.DialHTTP("tcp", "localhost:9005")
	if err != nil {
		log.Fatal(err)
	}

	// Insert required key-values
	inserts := []struct {
		k string
		v string
	}{
		{"Maria", "100"},
		{"John", "20"},
		{"Anna", "40"},
		{"Tim", "100"},
		{"Alex", "10"},
	}

	fmt.Println("=== Initial placement (after inserts) ===")
	for _, kv := range inserts {
		put(server, kv.k, kv.v)
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("%s -> %s\n", kv.k, getOwner(server, kv.k))
	}

	fmt.Println("\n=== After Node6 goes down ===")
	deleteNode(server, "Node6")

	keys2 := []string{"Anna", "Maria", "Lauren", "John", "Thomas"}
	for _, k := range keys2 {
		fmt.Printf("%s -> %s\n", k, getOwner(server, k))
	}
}

// -------------------- Main --------------------

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "demo" {
		dynamoDemo()
		return
	}
	main_client()
}

// -------------------- gossip client --------------------

func main_client() {
	rand.Seed(time.Now().UnixNano())
	Z_TIME := rand.Intn(Z_TIME_MAX-Z_TIME_MIN) + Z_TIME_MIN

	// Connect to RPC server
	server, _ := rpc.DialHTTP("tcp", "localhost:9005")

	args := os.Args[1:]

	// Get ID from command line argument
	if len(args) == 0 {
		fmt.Println("No args given")
		return
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("Found Error", err)
	}

	fmt.Println("Node", id, "will fail after", Z_TIME, "seconds")

	currTime := calcTime()
	// Construct self
	self_node = shared.Node{ID: id, Hbcounter: 0, Time: currTime, Alive: true}
	var self_node_response shared.Node // Allocate space for a response to overwrite this

	// Add node with input ID
	if err := server.Call("Membership.Add", self_node, &self_node_response); err != nil {
		fmt.Println("Error: Membership.Add()", err)
	} else {
		fmt.Printf("Success: Node created with id= %d\n", id)
	}

	// test out the get and the update
	var fetched_node shared.Node
	if err := server.Call("Membership.Get", self_node.ID, &fetched_node); err != nil {
		fmt.Println("Error fetching node: ", err)
	} else {
		fmt.Printf("Fetched node from server: %+v\n", fetched_node)
	}

	// increment the node's heartbeat to test update
	self_node.Hbcounter++
	var updatedNode shared.Node
	if err := server.Call("Membership.Update", self_node, &updatedNode); err != nil {
		fmt.Printf("Error updating node: %v\n", err)
	} else {
		fmt.Printf("Updated node on server: %+v\n", updatedNode)
	}

	neighbors := self_node.InitializeNeighbors(id)
	fmt.Println("Neighbors:", neighbors)

	membership := shared.NewMembership()
	membership.Add(self_node, &self_node)

	// add the node to the server's consistent hashing ring
	addArgs := shared.AddNodeArgs{Server: fmt.Sprintf("Node%d", id)}
	addReply := shared.AddNodeReply{}
	server.Call("KV.AddNode", &addArgs, &addReply)
	fmt.Printf("Node %d added to ring\n", id)

	time.AfterFunc(time.Second*X_TIME, func() { runAfterX(server, &self_node, &membership, id) })
	time.AfterFunc(time.Second*Y_TIME, func() { runAfterY(server, neighbors, &membership, id) })

	// Disable random crash for Dynamo assignment (keep if you want for gossip lab)
	// time.AfterFunc(time.Second*time.Duration(Z_TIME), func() { runAfterZ(server, id) })

	wg.Add(1)
	wg.Wait()
}

func runAfterX(server *rpc.Client, node *shared.Node, membership **shared.Membership, id int) {
	if node.Alive {
		node.Hbcounter++
		node.Time = calcTime()

		membershipMu.Lock()
		(*membership).Members[node.ID] = *node
		membershipMu.Unlock()

		var reply shared.Node
		server.Call("Membership.Update", *node, &reply)

		checkForPendingWrites(server, id)
	}

	time.AfterFunc(time.Second*X_TIME,
		func() { runAfterX(server, node, membership, id) })
}

func runAfterY(server *rpc.Client, neighbors [2]int, membership **shared.Membership, id int) {
	if self_node.Alive {
		// pick fresh random neighbors each round to gossip to
        neighbors = self_node.InitializeNeighbors(id)

		sendMessage(server, neighbors[0], *membership)
		sendMessage(server, neighbors[1], *membership)

		*membership = readMessages(server, id, *membership)

		checkForFailures(server, *membership);

		membershipMu.Lock()
		printMembership(**membership)
		membershipMu.Unlock()
	}

	time.AfterFunc(time.Second*Y_TIME,
		func() {
			runAfterY(server, neighbors, membership, id)
		})
}

func runAfterZ(server *rpc.Client, id int) {
	self_node.Alive = false
	self_node.Time = calcTime()

	// remove from ring
    delArgs := shared.DeleteNodeArgs{Server: fmt.Sprintf("Node%d", id)}
    delReply := shared.DeleteNodeReply{}
    server.Call("KV.DeleteNode", &delArgs, &delReply)

	var reply shared.Node
	if err := server.Call("Membership.Update", self_node, &reply); err != nil {
		fmt.Println("Error: Membership.Update() on crash", err)
	}
	fmt.Println("Node", id, "CRASHED")
}

func printMembership(m shared.Membership) {
	ids := make([]int, 0, len(m.Members))
	for id := range m.Members {
		ids = append(ids, id)
	}

	sort.Ints(ids)
	fmt.Println("<<<<<< MEMBERSHIP TABLE >>>>>>")
	for _, id := range ids {
		val := m.Members[id]
		status := "is Alive"
		if !val.Alive {
			status = "is Dead"
		}
		fmt.Printf("Node %d has hb %d, time %.1f and %s\n", val.ID, val.Hbcounter, val.Time, status)
	}
	fmt.Println("")
}

/* -------- For consensus kinda taken from Raft Assigment ---------- */

// checkForPendingWrites - called every second to check if any candidates are asking for votes
// if there s vote request, vote yes or no
func checkForPendingWrites(server *rpc.Client, id int) {
	if !self_node.Alive {
		return
	}

	var writeReq shared.WriteRequest
	server.Call("QuorumTracker.GetWriteRequest", id, &writeReq)

	if writeReq.Key == "" {
        return  // nothing pending
    }
	if votedWrites[writeReq.WriteID] {
		return
	}

	fmt.Printf("Node %d storing key=%s value=%s\n", id, writeReq.Key, writeReq.Value)

	ack := shared.WriteAck{
        NodeID: id,
		WriteID: writeReq.WriteID,
        Key:    writeReq.Key,
        Value:  writeReq.Value,
        Stored: true,
    }
    var ok bool
    server.Call("QuorumTracker.AckWrite", ack, &ok)
	votedWrites[writeReq.WriteID] = true

}

func checkForFailures(server *rpc.Client, membership *shared.Membership) {
	now := calcTime()

	membershipMu.Lock()
    defer membershipMu.Unlock()

	for id, node := range membership.Members {
		if id == self_node.ID {
			continue;
		}
		if node.Alive && (now - node.Time) > FAILURE_TIMEOUT {
			fmt.Printf("Node %d suspected FAILED (last seen %.1fs ago)\n", id, now-node.Time)
			membership.Members[id] = shared.Node {
				ID: id,
				Hbcounter: node.Hbcounter,
				Time: node.Time,
				Alive: false,
			}
		}
	}
}
