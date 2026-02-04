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
)

const (
	MAX_NODES  = 8
	X_TIME     = 1
	Y_TIME     = 2
	Z_TIME_MAX = 100
	Z_TIME_MIN = 10
)

var self_node shared.Node

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
	self_node = shared.Node{ID: id, Hbcounter: 0, Time: currTime, Alive: true}
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

	time.AfterFunc(time.Second*X_TIME, func() { runAfterX(server, &self_node, &membership, id) })
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
		(*membership).Members[node.ID] = *node

		var reply shared.Node
		server.Call("Membership.Update", *node, &reply)
	}

	time.AfterFunc(time.Second*X_TIME,
		func() { runAfterX(server, node, membership, id) })
}

func runAfterY(server *rpc.Client, neighbors [2]int, membership **shared.Membership, id int) {
	//TODO
	if self_node.Alive {
		sendMessage(server, neighbors[0], *membership)
		sendMessage(server, neighbors[1], *membership)

		*membership = readMessages(server, id, *membership)

		printMembership(**membership)
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
}

func printMembership(m shared.Membership) {
	for _, val := range m.Members {
		status := "is Alive"
		if !val.Alive {
			status = "is Dead"
		}
		fmt.Printf("Node %d has hb %d, time %.1f and %s\n", val.ID, val.Hbcounter, val.Time, status)
	}
	fmt.Println("")
}
