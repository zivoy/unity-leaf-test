package main

import (
	"context"
	pb "exampleMulti/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
	"time"
)

func TestNETServer_Conn(t *testing.T) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.Dial(port, opts)
	if err != nil {
		t.Error("could not connect to server: ", err)
	}
	defer conn.Close()
}

func TestNETServer_Request(t *testing.T) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials())
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("Test #%d", i), func(t *testing.T) {
			conn, err := grpc.Dial(port, opts)
			if err != nil {
				t.Error("could not connect to TCP server: ", err)
			}
			defer conn.Close()
			c := pb.NewColourGeneratorClient(conn)
			cc := "001100"

			ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
			col, err := c.GetRandColour(ctx, &pb.CurrentColour{Colour: cc})
			if err != nil {
				t.Error("error on response: ", err)
			}
			fmt.Println(col.Colour, "#"+cc)
		})
	}
}
