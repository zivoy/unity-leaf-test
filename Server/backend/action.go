package backend

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

// utility functions for keeping actions in time
func (g *Game) checkLastActionTime(actionKey string, created time.Time) bool {
	lastAction, _ := g.lastAction[actionKey]
	return lastAction.Add(ActionThrottle).Before(created)
}
func (g *Game) updateLastActionTime(actionKey string, created time.Time) {
	if actionKey != "" {
		g.lastAction[actionKey] = created
	}
}

type Action interface {
	Perform(game *Game) Change
	UpdateAction(game *Game)
	Runnable(game *Game) bool
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

func (b *baseAction) ActionCode(g *Game) string {
	entity := g.Entities[b.EntityID()]
	if entity == nil {
		return ""
	}
	return fmt.Sprintf("%T:%s", b, entity.ID.String())
}

func (b *baseAction) UpdateAction(g *Game) {
	g.updateLastActionTime(b.ActionCode(g), b.Created)
}
func (b *baseAction) Runnable(g *Game) bool {
	return g.checkLastActionTime(b.ActionCode(g), b.Created)
}

type MoveAction struct {
	baseAction
	Position Coordinate
}

func (m *MoveAction) Perform(game *Game) Change {
	entity := game.Entities[m.EntityID()]
	if entity == nil {
		return nil
	}

	pos := m.Position
	entity.Set(pos)
	// Inform the client that the entity moved.
	change := &MoveChange{
		Position:  &pos,
		baseEvent: m.baseEvent,
	}
	return change
}

type RemoveAction struct {
	baseAction
	*Entity
}

func (r *RemoveAction) Perform(game *Game) Change {
	delete(game.Entities, r.EntityID())
	change := &RemoveEntityChange{
		baseEvent: r.baseEvent,
	}
	return change
}

type AddAction struct {
	baseAction
	*Entity
	ClientID           Token
	RemoveOnDisconnect bool
}

func (a *AddAction) Perform(game *Game) Change {
	game.Entities[a.EntityID()] = a.Entity
	if a.RemoveOnDisconnect {
		game.ownedEntities[a.ClientID] = append(game.ownedEntities[a.ClientID], a.EntityID())
	}
	change := &AddEntityChange{
		baseEvent: a.baseEvent,
		Entity:    a.Entity,
	}
	return change
}
