package backend

import "image/color"

// Player contains information unique to local and remote players.
type Player struct {
	IdentifierBase
	Positioner
	CurrentPosition Coordinate
	Name            string
	Colour          color.Color
}

// Position determines the player position.
func (p *Player) Position() Coordinate {
	return p.CurrentPosition
}

// Set sets the position of the player.
func (p *Player) Set(c Coordinate) {
	p.CurrentPosition = c
}
