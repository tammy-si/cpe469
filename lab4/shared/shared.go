package shared

import (
	"fmt"
	"math/rand"
	"time"
	"sync"
)

const (
	MAX_NODES = 8
)

// Node struct represents a computing node.
type Node struct {
	ID        int
	Hbcounter int
	Time      float64
	Alive     bool
	// added fields for Raft
	State	  int // 0 = Follower, 1 = Candidate, 2 = Leader
	CurrentTerm int
	VotedFor  int  // -1 means hasn't voted yet this term
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

/* ------ STUFF FOR RAFT VOTING -------- */

// Vote Request for Raft election
// represents a candidation asking for votes
type VoteRequest struct {
	CandidateID int 	// which node is asking for votes
	Term		int 	// which election term this is for
}

// VoteResponse represents one node's vote (Yes or No) for a candidate
// each node that sees a vote requests creates one of these to say yes or no
type VoteResponse struct {
	NodeID	int		// which node is casting the vote
	CandidateID int // who they're voting for
	Term	int 	// which term they're voting in
	VoteGranted bool	// true = yes, false = no
}

type Votes struct {
	mu	sync.Mutex
	Requests map[int]VoteRequest		// candidateID -> their vote request. Store when candidate asks for votes
	Responses map[int][]VoteResponse	// candidatID -> all the votes that candidate received
}

func NewVotes() *Votes {
	return &Votes{
		Requests: make(map[int]VoteRequest),
		Responses: make(map[int][]VoteResponse),
	}
}

// RequestVote is called by a candidate to post their vote request on the server
func (v *Votes) RequestVote(req VoteRequest, reply *bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	// store vote request so others to see it
	// key is candidate id
	v.Requests[req.CandidateID] = req
	if reply != nil {
		*reply = true
	}
	return nil
}

// GetVoteRequest - called by nodes to check if anyone's asking for votes
// checks to see if there's election happening
func (v *Votes) GetVoteRequest(NodeID int, req *VoteRequest) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// look through all the pending vote requests
	for _, voteReq := range v.Requests {
		*req = voteReq
		return nil
	}

	// no vote requests found
	req.CandidateID = -1	// -1 to signal no 
	return nil
}

// CastVote - called by a node to submit their vote for a candidate
func (v *Votes) CastVote(vote VoteResponse, reply *bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// find which candidate this vote is for by matching the term
	candidateID := vote.CandidateID
    v.Responses[candidateID] = append(v.Responses[candidateID], vote)

	if reply != nil {
		*reply = true
	}

	return nil
}

// CollectVotes - called by a candidate to retrieve all votes cast for them
func (v *Votes) CollectVotes(candidateID int, votes* []VoteResponse) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Responses[candidateID] != nil {
		// return all the votes for this candidate
		*votes = v.Responses[candidateID]
		// clear out the candidates vote responses, as they've been counted
		delete(v.Responses, candidateID)
	} else {
		// no votes received yet
		*votes = []VoteResponse{}
	}
	return nil
}
