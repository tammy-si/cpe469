package main

import (
	"fmt"
	"io"
	"lab3/shared"
	"net/http"
	"net/rpc"
)

func main() {
	main_server()
}

func main_server() {
	// create a Membership list
	nodes := shared.NewMembership()
	requests := shared.NewRequests()

	// register nodes with `rpc.DefaultServer`
	rpc.Register(nodes)
	rpc.Register(requests)

	// register an HTTP handler for RPC communication
	rpc.HandleHTTP()

	// sample test endpoint
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "RPC SERVER LIVE!")
	})

	// listen and serve default HTTP server
	http.ListenAndServe("localhost:9005", nil)

	fmt.Println(nodes, requests)
}
