package main

import (
	"io"
	"lab3/shared"
	"log"
	"net/http"
	"net/rpc"
)

func main() {
	nodes := shared.NewMembership()
	requests := shared.NewRequests()
	votes := shared.NewVotes() 
	
	if err := rpc.Register(nodes); err != nil {
		log.Fatal(err)
	}
	if err := rpc.Register(requests); err != nil {
		log.Fatal(err)
	}

	if err := rpc.Register(votes); err != nil {
		log.Fatal(err)
	}

	rpc.HandleHTTP()

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "RPC SERVER LIVE!")
	})

	log.Println("RPC server listening on localhost:9005")
	log.Fatal(http.ListenAndServe("localhost:9005", nil))
}
