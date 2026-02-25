package main

import (
	"io"
	"lab3/shared"
	"log"
	"net/http"
	"net/rpc"
	"sync"
)

type KV struct {
	mu sync.Mutex
	data map[string]string
}

func (kv *KV) Get (args *shared.GetArgs, reply *shared.GetReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	reply.Value = kv.data[args.Key]
	return nil
}

func (kv *KV) Put(args *shared.PutArgs, reply *shared.PutReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.data[args.Key] = args.Value
	return nil
}

func NewKV() *KV {
	//TODO
	return &KV{
		data: make(map[string]string),
	}
}

func main() {
	nodes := shared.NewMembership()
	requests := shared.NewRequests()
	kv := NewKV()

	if err := rpc.Register(nodes); err != nil {
		log.Fatal(err)
	}
	if err := rpc.Register(requests); err != nil {
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

