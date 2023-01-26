package main

import (
	pb "exampleMulti/proto"
	"google.golang.org/grpc"
	"log"
	"net"
)

const (
	port = ":50051"
)

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen to port [%s]: %v", port, err)
	}
	s := grpc.NewServer()
	pb.RegisterColourGeneratorServer(s, &pb.Server{})
	if err = s.Serve(lis); err != nil {
		log.Fatalln("Failed to start the server:", err)
	}
}
