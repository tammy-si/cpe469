package main

import (
	"fmt"
	"net/rpc"
	"strings"
	"os"
	"time"
	"bufio"
)

type Args struct {
	Text, Term string
}

func main() {
	// connect to the server at localhost:8080
	client, err := rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting: ", err)
		return
	}
	defer client.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Input searched word: ")
	term, _ := reader.ReadString('\n')
	term = strings.TrimSpace(term)
	if term == "" {
		fmt.Println("Empty search term.")
		return
	}

	data, err := os.ReadFile("../war_and_peace.html")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	text := string(data)

	args := Args{Text: text, Term: term}
	var result int

	start := time.Now()
	err = client.Call("StringCounter.SequentialCount", args, &result)
	elasped := time.Since(start)

	if err != nil {
		fmt.Println("Error calling StringCounter.SequentialCount:", err)
		return
	}

	fmt.Printf("Result of string count with rpc: %d. It took %v.\n", result, elasped)

	var go_routine_result int
	go_rountine_start := time.Now()
	err = client.Call("StringCounter.ConcurrentCount", args, &go_routine_result)
	go_rountine_elasped := time.Since(go_rountine_start)

	if err != nil {
		fmt.Println("Error calling StringCounter.ConcurrentCount:", err)
		return
	}

	fmt.Printf("Result of string count with rpc and 8 go routines: %d. It took %v.\n", go_routine_result, go_rountine_elasped)
}
