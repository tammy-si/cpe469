package main

import (
	"fmt"
	"net"
	"net/rpc"
	"lab2/search_functions"
)

type StringCounter struct {}

type Args struct {
	Text, Term string
}

func (s *StringCounter) SequentialCount(args *Args, reply * int) error {
	text := args.Text
	term := args.Term
	if len(term) == 0 || len(term) > len(text) {
		*reply = 0
		return nil
	}
	*reply = search_functions.CountSequential(text, term)
	return nil
}

func (s *StringCounter) ConcurrentCount(args *Args, reply * int) error {
	text := args.Text
	term := args.Term
	if len(term) == 0 || len(term) > len(text) {
		*reply = 0
		return nil
	}
	*reply = search_functions.CountConcurrent(text, term, 8)
	return nil
}

func main() {
	sc := new(StringCounter)
	rpc.Register(sc)

	// listening on pport 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
        fmt.Println("Error listening:", err)
        return
    }
	fmt.Println("Server is listening on port 8080...")
	rpc.Accept(listener)
}
