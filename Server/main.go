package main

import (
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

var (
	port = flag.Int("port", 50051, "port number for server")
	addr string
)

func init() {
	flag.Parse()
	addr = fmt.Sprintf(":%d", *port)
}

func main() {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen to port [%s]: %v", addr, err)
	}

	game := backend.NewGame()
	game.Start()
	server := NewGameServer(game)

	s := grpc.NewServer()
	pb.RegisterColourGeneratorServer(s, &RandomiseColour{})
	pb.RegisterGameServer(s, server)

	if err = s.Serve(lis); err != nil {
		log.Fatalln("Failed to start the server:", err)
	}
}
