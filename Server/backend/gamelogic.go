package backend

import (
	"github.com/google/uuid"
	"sync"
)

type Game struct {
	Mu       sync.RWMutex
	Entities map[uuid.UUID]Identifier
}

type Identifier interface {
	ID() uuid.UUID
}

func (g *Game) RemoveEntity(id uuid.UUID) {
	delete(g.Entities, id)
}
