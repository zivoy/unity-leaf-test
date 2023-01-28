package backend

import "image/color"

// Player contains information unique to local and remote players.
type Entity struct {
	IdentifierBase
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
