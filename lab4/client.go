package main

import (
	"fmt"
	"lab3/shared"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
	"sort"
)

const (
	MAX_NODES  = 8
	X_TIME     = 1
	Y_TIME     = 2
	Z_TIME_MAX = 100
	Z_TIME_MIN = 10
	ELECTION_MIN = 150		// the node's timeout Y will be between the election min and max
	ELECTION_MAX   = 300
	VOTE_WAIT_TIME = 300
	HEARTBEAT_INTERVAL = 50
)

var self_node shared.Node
var membershipMu sync.Mutex
var electionTimer *time.Timer // Holds reference to the currently scheduled election timer

// Send the current membership table to a neighboring node with the provided ID
func sendMessage(server *rpc.Client, id int, membership *shared.Membership) {
	//TODO
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
		fmt.Println("Error: Requests.Add()", err)
		return membership
	} else {
		merged := shared.CombineTables(membership, &incoming)
		return merged
	}
	//TODO
}

func calcTime() float64 {
	return float64(time.Now().UnixNano()) / 1e9
	//TODO
}

var wg = &sync.WaitGroup{}

func main() {
	main_client()
}

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
	self_node = shared.Node{ID: id, Hbcounter: 0, Time: currTime, Alive: true, State: 0, CurrentTerm: 0, VotedFor: -1}
	var self_node_response shared.Node // Allocate space for a response to overwrite this

	// Add node with input ID
	if err := server.Call("Membership.Add", self_node, &self_node_response); err != nil {
		fmt.Println("Error:2 Membership.Add()", err)
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
		fmt.Printf("Error fetching node: ", err)
	} else {
		fmt.Printf("Updated node on server: %+v\n", updatedNode)
	}

	// testing out the request stuff (listen and combineTables)
	reqs := shared.NewRequests()

	// fake membership for node1
	m1 := shared.NewMembership()
	node1 := shared.Node{ID: 1, Hbcounter: 5, Time: 1.0, Alive: true}
	m1.Add(node1, &node1)

	reqs.Pending[1] = *m1

	var reply shared.Membership
	err = reqs.Listen(1, &reply)
	if err != nil {
		fmt.Println("Listen error: ", err)
	} else {
		fmt.Println("Listen returned membership for node 1: ")
		for _, n := range reply.Members {
			fmt.Printf("Node %d hb=%d, alive=%v\n", n.ID, n.Hbcounter, n.Alive)
		}
	}

	// nothing pending
	var emptyReply shared.Membership
	err = reqs.Listen(1, &emptyReply)
	fmt.Println("Listen returned empty membership when nothing pending:")
	fmt.Printf("Members map size: %d\n", len(emptyReply.Members))

	neighbors := self_node.InitializeNeighbors(id)
	fmt.Println("Neighbors:", neighbors)

	membership := shared.NewMembership()
	membership.Add(self_node, &self_node)

	sendMessage(server, neighbors[0], membership)

	//crashTime := self_node.CrashTime()

	// to start election timer
	scheduleElectionTimeout(server, &membership, id)

	time.AfterFunc(X_TIME * time.Millisecond, func() { runAfterX(server, &self_node, &membership, id) })
	time.AfterFunc(time.Second*Y_TIME, func() { runAfterY(server, neighbors, &membership, id) })
	time.AfterFunc(time.Second*time.Duration(Z_TIME), func() { runAfterZ(server, id) })

	wg.Add(1)
	wg.Wait()
}

func runAfterX(server *rpc.Client, node *shared.Node, membership **shared.Membership, id int) {
	//TODO
	if node.Alive {
		node.Hbcounter++
		node.Time = calcTime()

		//
		membershipMu.Lock()
		(*membership).Members[node.ID] = *node
		membershipMu.Unlock()

		var reply shared.Node
		server.Call("Membership.Update", *node, &reply)

		// check if anyone's asking for votes
		checkForVoteRequests(server, membership, id)

		// check for heartbeats from leader
		checkForHeartbeat(server, membership, id)
	}

	time.AfterFunc(X_TIME * time.Millisecond,
		func() { runAfterX(server, node, membership, id) })
}

func runAfterY(server *rpc.Client, neighbors [2]int, membership **shared.Membership, id int) {
	//TODO
	if self_node.Alive {
		sendMessage(server, neighbors[0], *membership)
		sendMessage(server, neighbors[1], *membership)

		*membership = readMessages(server, id, *membership)

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
	//TODO
	self_node.Alive = false
	self_node.Time = calcTime()

	var reply shared.Node
	if err := server.Call("Membership.Update", self_node, &reply); err != nil {
		fmt.Println("Error: Membership.Update() on crash", err)
	}
	fmt.Println("Node", id, "CRASHED")
	if electionTimer != nil {
		electionTimer.Stop()
	}

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

		// Print raft state info
		stateStr := "FOLLOWER"
		if val.State == 1 {
			stateStr = "CANDIDATE"
		} else if val.State == 2 {
			stateStr = "LEADER"
		}
		fmt.Printf("Node %d has hb %d, time %.1f and %s. State: %s, Term: %d\n", val.ID, val.Hbcounter, val.Time, status, stateStr, val.CurrentTerm)
	}
	fmt.Println("")
}

// schedule election timeout with a random delay
// timer that will call handleElectionTimeout after random delay can be stopped by stopping election Timer
func scheduleElectionTimeout(server *rpc.Client, membership **shared.Membership, id int) {
	// no scheudle if dead or leader
	if !self_node.Alive || self_node.State == 2{
		return
	}

	if electionTimer != nil {
        electionTimer.Stop()
    }

	// each node needs a different random timeout so that there are no ties
	timeout := time.Duration(ELECTION_MIN+rand.Intn(ELECTION_MAX-ELECTION_MIN)) * time.Millisecond

	electionTimer = time.AfterFunc(timeout, func() {
		handleElectionTimeout(server, membership, id)
	})
}


// handleElectionTimeout - Called when election timer expires without hearing from leader
func handleElectionTimeout(server *rpc.Client, membership **shared.Membership, id int) {
	if !self_node.Alive || self_node.State == 2 {
		return
	}

	// become a candidate, increment term, and vote for self
	self_node.State = 1
	self_node.CurrentTerm++
	self_node.VotedFor = self_node.ID
	fmt.Printf(">>> Node %d became CANDIDATE for term %d (NO HEARTBEAT RECEIVED)\n", id, self_node.CurrentTerm)

	// just updates the membership state
	var reply shared.Node
	server.Call("Membership.Update", self_node, &reply)

	// request votes from other nodes
	voteReq := shared.VoteRequest {
		CandidateID: id,
		Term: self_node.CurrentTerm,
	}

	for nodeID := 1; nodeID <= MAX_NODES; nodeID++ {
		if nodeID != id {
			var ok bool
			server.Call("Votes.RequestVote", voteReq, &ok)
		}
	}

	// candidates waits 300 ms for votes to come in, then count them
	time.AfterFunc(VOTE_WAIT_TIME*time.Millisecond, func () {
		countVotesAndDecide(server, membership, id)
	})
}

// candidate counting the votes they got, collects votes and checks if there's a majority
func countVotesAndDecide(server *rpc.Client, membership **shared.Membership, id int) {
	if self_node.State != 1 {
		return
	}

	var votes []shared.VoteResponse
	server.Call("Votes.CollectVotes", id, &votes)

	voteCount := 1
	for _, vote := range votes {
		if vote.VoteGranted && vote.Term == self_node.CurrentTerm && vote.CandidateID == id {
			voteCount++
		}
	}

	// check for majority
	majority := (MAX_NODES / 2) + 1

	if voteCount >= majority {
		self_node.State = 2
		fmt.Printf("Node %d is now LEADER for term %d\n", id, self_node.CurrentTerm)

		// update state
		var reply shared.Node
		server.Call("Membership.Update", self_node, &reply)

		// stop electionTimer for leader
		if electionTimer != nil {
			electionTimer.Stop()
		}
		sendHeartbeat(server, id)
	} else {
		//lost election or split vote
		self_node.State = 0    
		self_node.VotedFor = -1 

		var reply shared.Node
		server.Call("Membership.Update", self_node, &reply)

		scheduleElectionTimeout(server, membership, id)
	}
}

// checkForVoteRequests - called every second to check if any candidates are asking for votes
// if there s vote request, vote yes or no
func checkForVoteRequests(server *rpc.Client, membership **shared.Membership, id int) {
	if !self_node.Alive {
		return
	}

	var voteReq shared.VoteRequest
	server.Call("Votes.GetVoteRequest", id, &voteReq)

	// no vote requests found = -1
	if voteReq.CandidateID != -1 || voteReq.CandidateID == self_node.ID{
		// there is a vote request
		grantVote := false
		if voteReq.Term > self_node.CurrentTerm {
			self_node.CurrentTerm = voteReq.Term
			self_node.State = 0
			self_node.VotedFor = -1

			//reset election timer
			resetElectionTimer(server, membership, id)
		}
		// if haven't voted or new term
		if voteReq.Term == self_node.CurrentTerm &&
		(self_node.VotedFor == -1 || self_node.VotedFor == voteReq.CandidateID) {
			grantVote = true
			self_node.VotedFor = voteReq.CandidateID

			fmt.Printf("Node %d VOTED FOR candidate %d in term %d\n",
						id, voteReq.CandidateID, voteReq.Term)
			// If I voted for someone else, reset my election timer
			if voteReq.CandidateID != self_node.ID {
				resetElectionTimer(server, membership, id)
			}

		} else {
			fmt.Printf("Node %d DENIED vote to candidate %d (already voted for %d in term %d)\n",
			id, voteReq.CandidateID, self_node.VotedFor, self_node.CurrentTerm)
		}

		// vote send voteResponse
		vote := shared.VoteResponse {
			NodeID: id,
			CandidateID: voteReq.CandidateID,
			Term: voteReq.Term,
			VoteGranted: grantVote,
		}

		var ok bool
		server.Call("Votes.CastVote", vote, &ok)

		var reply shared.Node
		server.Call("Membership.Update", self_node, &reply)
	}
}

// called by leader to tell follower to say they're still leader
func sendHeartbeat(server *rpc.Client, id int) {
	if !self_node.Alive || self_node.State != 2 {
		return
	}

	var ok bool
	server.Call("Votes.SendHeartbeat", self_node.CurrentTerm, &ok)

	// loop the heartbeat every second
	time.AfterFunc(HEARTBEAT_INTERVAL * time.Millisecond, func() {
			sendHeartbeat(server, id)
		})
}

// followers call this to check if leader sent a heartbeat
// restart the electionTimer if heartbeat seen in time
func checkForHeartbeat(server *rpc.Client, membership **shared.Membership, id int) {
	if !self_node.Alive {
		return
	}
	var term int
	server.Call("Votes.CheckHeartbeat", id, &term)

	if term == -1 {
		// don't reset the election timer
		return
	}

	if self_node.State == 2 {
		// leader does not reset timer
		return
	}

	if self_node.State != 0 {
		self_node.State = 0 // step down if we were candidate
	}

	if term > self_node.CurrentTerm {
		self_node.CurrentTerm = term
		self_node.VotedFor = -1
	}
	resetElectionTimer(server, membership, id)

}

func resetElectionTimer(server *rpc.Client, membership **shared.Membership, id int) {
	if electionTimer != nil {
		electionTimer.Stop()
	}
	scheduleElectionTimeout(server, membership, id)
}
