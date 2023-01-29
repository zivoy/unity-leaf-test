package main

import (
	"context"
	pb "exampleMulti/proto"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
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
	rand.Seed(time.Now().Unix())
	server := NewGameServer()

	s := grpc.NewServer()
	pb.RegisterGameServer(s, server)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Starting server")
		if err = s.Serve(lis); err != nil {
			log.Fatalln("Failed to start the server:", err)
		}
		cancel()
	}()

	select {
	case <-signalChan:
	case <-ctx.Done():
	}
	server.Stop()
	log.Println("Shutting down")
	time.Sleep(500 * time.Millisecond)
}
