package shared

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	MAX_NODES = 8
	ACK_WAIT_TIME = 300
)

// Node struct represents a computing node.
type Node struct {
	ID        int
	Hbcounter int
	Time      float64
	Alive     bool
}

// Generate random crash time from 10-60 seconds
func (n Node) CrashTime() int {
	rand.Seed(time.Now().UnixNano())
	max := 60
	min := 10
	return rand.Intn(max-min) + min
}

func (n Node) InitializeNeighbors(id int) [2]int {
	neighbor1 := RandInt()
	for neighbor1 == id {
		neighbor1 = RandInt()
	}
	neighbor2 := RandInt()
	for neighbor1 == neighbor2 || neighbor2 == id {
		neighbor2 = RandInt()
	}
	return [2]int{neighbor1, neighbor2}
}

func RandInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(MAX_NODES-1+1) + 1
}

/*---------------*/

// Membership struct represents participanting nodes
type Membership struct {
	mu      sync.Mutex
	Members map[int]Node
}

// Returns a new instance of a Membership (pointer).
func NewMembership() *Membership {
	return &Membership{
		Members: make(map[int]Node),
	}
}

// Adds a node to the membership list.
func (m *Membership) Add(payload Node, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// add node to the Members hashmap, key is Node's id
	m.Members[payload.ID] = payload
	if reply != nil {
		*reply = payload
	}

	return nil
}

// Updates a node in the membership list.
func (m *Membership) Update(payload Node, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.Members[payload.ID]
	if !ok {
		return fmt.Errorf("node %d not found", payload.ID)
	}
	m.Members[payload.ID] = payload
	if reply != nil {
		*reply = payload
	}
	return nil
}

// Returns a node with specific ID.
func (m *Membership) Get(payload int, reply *Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.Members[payload]
	if !ok {
		return fmt.Errorf("node %d not found", payload)
	}
	if reply != nil {
		*reply = val
	}
	return nil
	//TODO
}

/*---------------*/

// Request struct represents a new message request to a client
type Request struct {
	ID    int
	Table Membership
}

// Requests struct represents pending message requests
type Requests struct {
	mu      sync.Mutex
	Pending map[int]Membership
}

// Returns a new instance of a Membership (pointer).
func NewRequests() *Requests {
	//TODO
	return &Requests{
		Pending: make(map[int]Membership),
	}
}

// Adds a new message request to the pending list
func (req *Requests) Add(payload Request, reply *bool) error {
	req.mu.Lock()
	defer req.mu.Unlock()

	req.Pending[payload.ID] = payload.Table
	if reply != nil {
		*reply = true
	}
	return nil
}

// Listens to communication from neighboring nodes.
func (req *Requests) Listen(ID int, reply *Membership) error {
	req.mu.Lock()
	defer req.mu.Unlock()

	// check if there's something pending for that id
	table, ok := req.Pending[ID]
	if !ok {
		// nothing pending so return empty membership
		*reply = Membership{Members: make(map[int]Node)}
		return nil
	}
	// there is new membership table pending, return it
	*reply = table
	delete(req.Pending, ID)
	return nil
}

func CombineTables(table1 *Membership, table2 *Membership) *Membership {
	result := NewMembership()

	table1.mu.Lock()
	// copy table1's stuff into the new combined membership
	for id, node := range table1.Members {
		result.Members[id] = node
	}
	table1.mu.Unlock()

	table2.mu.Lock()
	// now loop through with table2 and combine
	for id, node2 := range table2.Members {
		node1, ok := result.Members[id]
		// if this table2 has new id or updated version of the node
		if !ok || node2.Hbcounter > node1.Hbcounter {
			result.Members[id] = node2
		}
	}
	table2.mu.Unlock()

	return result
}

// Common RPC request/reply definitions
type PutArgs struct {
	Key   string
	Value string
}

type PutReply struct {
}

type GetArgs struct {
	Key string
}

type GetReply struct {
	Value string
}

type AddNodeArgs struct {
	Server string
}
type AddNodeReply struct{}

type DeleteNodeArgs struct {
	Server string
}
type DeleteNodeReply struct{}

type OwnerArgs struct {
	Key string
}
type OwnerReply struct {
	Server string
}

/* ------ STUFF FOR RAFT VOTING -------- */

// Vote Request for Raft election
// represents a candidation asking for QuorumTracker
type WriteRequest struct {
	WriteID string
	Key string 	// which node is asking for QuorumTracker
	Value		string 	// which election value this is for
}

// WriteAck represents one node's vote (Yes or No) for a candidate
// each node that sees a vote requests creates one of these to say yes or no
type WriteAck struct {
	NodeID	int		// which node is casting the vote
	WriteID string
	Key    string // who they're voting for
	Value  string 	// which value they're voting in
	Stored bool	// true = yes, false = no
}

type QuorumTracker struct {
	mu	sync.Mutex
	Requests map[string]WriteRequest		// key -> their vote request. Store when candidate asks for QuorumTracker
	Responses map[string][]WriteAck	// candidatID -> all the QuorumTracker that candidate received
}

func NewQuorumTracker() *QuorumTracker {
	return &QuorumTracker{
		Requests: make(map[string]WriteRequest),
		Responses: make(map[string][]WriteAck),
	}
}

// ProposeWrite is called by a candidate to post their vote request on the server
func (v *QuorumTracker) ProposeWrite(req WriteRequest, reply *bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	// store vote request so others to see it
	// key is candidate id
	v.Requests[req.WriteID] = req
	if reply != nil {
		*reply = true
	}
	return nil
}

// GetWriteRequest - called by nodes to check if anyone's asking for QuorumTracker
// checks to see if there's election happening
func (v *QuorumTracker) GetWriteRequest(NodeID int, req *WriteRequest) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// look through all the pending vote requests
	for _, voteReq := range v.Requests {
		*req = voteReq
		return nil
	}

	// no vote requests found
	req.Key = ""	// empty to say no request to write yet 
	return nil
}

// AckWrite - called by a node to submit their vote for a candidate
func (v *QuorumTracker) AckWrite(vote WriteAck, reply *bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// find which candidate this vote is for by matching the value
	id := vote.WriteID
	v.Responses[id] = append(v.Responses[id], vote)

	if reply != nil {
		*reply = true
	}

	return nil
}

// CollectQuorumTracker - called by a candidate to retrieve all QuorumTracker cast for them
func (v *QuorumTracker) CollectAcks(key string, QuorumTracker* []WriteAck) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Responses[key] != nil {
		// return all the QuorumTracker for this candidate
		*QuorumTracker = v.Responses[key]
		// clear out the candidates vote responses, as they've been counted
		delete(v.Responses, key)
		delete(v.Requests, key)
	} else {
		// no QuorumTracker received yet
		*QuorumTracker = []WriteAck{}
	}
	delete(v.Requests, key)
	return nil
}
