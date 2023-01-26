package proto

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
)

const (
	colourByte = 3
)

type Server struct{}

func (s Server) GetRandColour(ctx context.Context, curr *CurrentColour) (*NewColour, error) {
	hex := "#" + randomHex()
	log.Printf("Client's current colour: [#%s] sending [%s]", curr.Colour, hex)
	return &NewColour{Colour: hex}, nil
}

func (s Server) mustEmbedUnimplementedColourGeneratorServer() {}

func randomHex() string {
	bytes := make([]byte, colourByte)
	if _, err := rand.Read(bytes); err != nil {
		log.Panicln("Error generating random hex value", err)
	}
	return fmt.Sprintf("%X", bytes)
}
