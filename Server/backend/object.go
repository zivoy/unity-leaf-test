package backend

import (
	pb "exampleMulti/proto"
	"github.com/google/uuid"
	"image/color"
)

type Entity struct {
	ID              uuid.UUID
	CurrentPosition Coordinate
	Name            string
	Colour          color.Color
	Type            string
}

// Position determines the player position.
func (e *Entity) Position() Coordinate {
	return e.CurrentPosition
}

// Set sets the position of the player.
func (e *Entity) Set(c Coordinate) {
	e.CurrentPosition = c
}

func (e *Entity) ToProto() *pb.Entity {
	return &pb.Entity{
		Id:       e.ID.String(),
		Name:     e.Name,
		Colour:   colourToHex(e.Colour),
		Position: e.Position().ToProto(),
		Type:     e.Type,
	}
}

func EntityFromProto(entity *pb.Entity) (*Entity, error) {
	id, err := uuid.Parse(entity.GetId())
	if err != nil {
		return nil, err
	}
	return &Entity{
		ID:              id,
		CurrentPosition: Coordinate{},
		Name:            entity.GetName(),
		Colour:          colourFromHex(entity.GetColour()),
		Type:            entity.GetType(),
	}, nil
}
