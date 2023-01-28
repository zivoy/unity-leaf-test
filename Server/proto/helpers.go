package proto

import (
	"exampleMulti/backend"
	"fmt"
	"github.com/google/uuid"
)

func GetProtoEntity(entity *backend.Entity) *Entity {
	r, g, b, _ := entity.Colour.RGBA()
	col := fmt.Sprintf("#%02x%02x%02x", r&0xff, g&0xff, b&0xff)
	return &Entity{
		Id:       entity.ID().String(),
		Name:     entity.Name,
		Colour:   col,
		Position: GetProtoCoordinate(entity.Position()),
		Type:     entity.Type,
	}
}

func GetProtoCoordinate(coordinate backend.Coordinate) *Position {
	return &Position{
		X: float32(coordinate.X),
		Y: float32(coordinate.Y),
	}
}

func GetBackendCoordinate(position *Position) backend.Coordinate {
	return backend.Coordinate{
		X: float64(position.X),
		Y: float64(position.Y),
	}
}

func ParseUUID(id string) (uuid.UUID, bool) {
	uid, err := uuid.Parse(id)
	return uid, err == nil
}
