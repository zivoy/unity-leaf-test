package backend

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

// utility functions for keeping actions in time
func (g *Game) checkLastActionTime(actionKey string, created time.Time) bool {
	lastAction, ok := g.lastAction[actionKey]
	return !(ok && lastAction.Sub(created) < ActionThrottle)
}
func (g *Game) updateLastActionTime(actionKey string, created time.Time) {
	g.lastAction[actionKey] = created
}

type Action interface {
	Perform(game *Game)
}

type baseAction struct {
	baseEvent
	Created time.Time
}

func (g *Game) getBaseAction(id uuid.UUID) baseAction {
	return baseAction{
		baseEvent: baseEvent{
			id:  id,
			gId: g.GameId,
		},
		Created: time.Now(),
	}
}

type MoveAction struct {
	baseAction
	Position Coordinate
}

func (m MoveAction) Perform(game *Game) {
	entity := game.Entities[m.EntityID()]
	if entity == nil {
		return
	}

	actionKey := fmt.Sprintf("%T:%s", m, entity.ID.String())
	if !game.checkLastActionTime(actionKey, m.Created) {
		return
	}
	position := entity.Position()
	pos := m.Position
	if d := position.Distance(pos); d > maxMove {
		dm := d / maxMove
		pos.Y /= dm
		pos.X /= dm
	}
	entity.Set(pos)
	// Inform the client that the entity moved.
	change := &UpdateEntityChange{
		Entity:    entity,
		baseEvent: m.baseEvent,
	}
	game.sendChange(change)
	game.updateLastActionTime(actionKey, m.Created)
}

type RemoveAction struct {
	baseAction
	*Entity
}

func (r *RemoveAction) Perform(game *Game) {
	game.Mu.Lock()
	delete(game.Entities, r.EntityID())
	game.Mu.Unlock()
	game.sendChange(&RemoveEntityChange{
		baseEvent: r.baseEvent,
	})
}

type AddAction struct {
	baseAction
	*Entity
	ClientID           uuid.UUID
	RemoveOnDisconnect bool
}

func (a *AddAction) Perform(game *Game) {
	game.Mu.Lock()
	game.Entities[a.EntityID()] = a.Entity
	if a.RemoveOnDisconnect {
		game.ownedEntities[a.ClientID] = append(game.ownedEntities[a.ClientID], a.EntityID())
	}
	game.Mu.Unlock()
	game.sendChange(&AddEntityChange{
		baseEvent: a.baseEvent,
		Entity:    a.Entity,
	})
}
