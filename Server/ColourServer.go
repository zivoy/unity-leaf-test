package main

import (
	"context"
	"crypto/rand"
	"exampleMulti/proto"
	"fmt"
	"image/color"
	"log"
)

const (
	colourByte = 3
)

type RandomiseColour struct {
	proto.UnimplementedColourGeneratorServer
}

func (s RandomiseColour) GetRandColour(ctx context.Context, curr *proto.CurrentColour) (*proto.NewColour, error) {
	hex := "#" + randomHex()
	log.Printf("Client's current colour: [#%s] sending [%s]", curr.Colour, hex)
	return &proto.NewColour{Colour: hex}, nil
}

func randomHex() string {
	return ToHex(randColour())
}

func ToHex(colour color.Color) string {
	r, g, b, _ := colour.RGBA()
	return fmt.Sprintf("%02x%02x%02x", r & 0xff, g& 0xff, b& 0xff)
}

func randColour() color.Color {
	bytes := make([]byte, colourByte)
	if _, err := rand.Read(bytes); err != nil {
		log.Panicln("Error generating random hex value", err)
	}
	return color.RGBA{R: bytes[0], G: bytes[1], B: bytes[2]}
}
