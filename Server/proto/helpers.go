package proto

import (
	"exampleMulti/backend"
	"fmt"
	"log"
)

func GetProtoEntity(entity backend.Identifier) *Entity {
	switch entity.(type) {
	case *backend.Player:
		player := entity.(*backend.Player)
		protoPlayer := Entity_Player{
			Player: GetProtoPlayer(player),
		}
		return &Entity{
			Entity: &protoPlayer,
			Id:     player.ID().String(),
		}
	}
	log.Printf("cannot get proto entity for %T -> %+v", entity, entity)
	return nil
}

func GetProtoPlayer(player *backend.Player) *Player {
	r, g, b, _ := player.Colour.RGBA()
	col := fmt.Sprintf("#%02x%02x%02x", r&0xff, g&0xff, b&0xff)
	return &Player{
		Name:     player.Name,
		Position: GetProtoCoordinate(player.Position()),
		Colour:   col,
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
