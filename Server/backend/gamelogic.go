package backend

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	moveThrottle = 100 * time.Millisecond
	maxMove      = 5
)

// Game is the backend engine for the game. It can be used regardless of how
// game data is rendered, or if a game server is being used.
type Game struct {
	Entities        map[uuid.UUID]*Entity
	Mu              sync.RWMutex
	ChangeChannel   chan Change
	ActionChannel   chan Action
	lastAction      map[string]time.Time
	IsAuthoritative bool
}

// NewGame constructs a new Game struct.
func NewGame() *Game {
	game := Game{
		Entities:        make(map[uuid.UUID]*Entity),
		ActionChannel:   make(chan Action, 1),
		lastAction:      make(map[string]time.Time),
		ChangeChannel:   make(chan Change, 1),
		IsAuthoritative: true,
	}
	return &game
}

// Start begins the main game loop, which waits for new actions and updates the
// game state occordinly.
func (game *Game) Start() {
	go game.watchActions()
}

// watchActions waits for new actions to come in and performs them.
func (game *Game) watchActions() {
	for {
		action := <-game.ActionChannel
		game.Mu.Lock()
		action.Perform(game)
		game.Mu.Unlock()
	}
}

// AddEntity adds an entity to the game.
func (game *Game) AddEntity(entity *Entity) {
	game.Entities[entity.ID()] = entity
}

// UpdateEntity updates an entity.
func (game *Game) UpdateEntity(entity *Entity) {
	game.Entities[entity.ID()] = entity
}

// GetEntity gets an entity from the game.
func (game *Game) GetEntity(id uuid.UUID) *Entity {
	return game.Entities[id]
}

// RemoveEntity removes an entity from the game.
func (game *Game) RemoveEntity(id uuid.UUID) {
	delete(game.Entities, id)
}

// checkLastActionTime checks the last time an action was performed.
func (game *Game) checkLastActionTime(actionKey string, created time.Time, throttle time.Duration) bool {
	lastAction, ok := game.lastAction[actionKey]
	if ok && lastAction.After(created.Add(-1*throttle)) {
		return false
	}
	return true
}

// updateLastActionTime sets the last action time.
// The actionKey should be unique to the action and the actor (entity).
func (game *Game) updateLastActionTime(actionKey string, created time.Time) {
	game.lastAction[actionKey] = created
}

// sendChange sends a change to the change channel.
func (game *Game) sendChange(change Change) {
	select {
	case game.ChangeChannel <- change:
	}
}

// Coordinate is used for all position-related variables.
type Coordinate struct {
	X float64
	Y float64
}

// Distance calculates the distance between two coordinates.
func (c1 Coordinate) Distance(c2 Coordinate) float64 {
	return math.Sqrt(math.Pow(c2.X-c1.X, 2) + math.Pow(c2.Y-c1.Y, 2))
}

// IdentifierBase is embedded to satisfy the Identifier interface.
type IdentifierBase struct {
	UUID uuid.UUID
}

// ID returns the UUID of an entity.
func (e IdentifierBase) ID() uuid.UUID {
	return e.UUID
}

// Change is sent by the game engine in response to Actions.
type Change interface{}

// MoveChange is sent when the game engine moves an entity.
type MoveChange struct {
	Change
	*Entity
	Position Coordinate
}

// AddEntityChange occurs when an entity is added in response to an action.
// Currently this is only used for new lasers and players joining the game.
type AddEntityChange struct {
	Change
	*Entity
}

// RemoveEntityChange occurs when an entity has been removed from the game.
type RemoveEntityChange struct {
	Change
	*Entity
}

// Action is sent by the client when attempting to change game state. The
// engine can choose to reject Actions if they are invalid or performed too
// frequently.
type Action interface {
	Perform(game *Game)
}

// MoveAction is sent when a user presses an arrow key.
type MoveAction struct {
	Position Coordinate
	ID       uuid.UUID
	Created  time.Time
}

// Perform contains backend logic required to move an entity.
func (action MoveAction) Perform(game *Game) {
	entity := game.GetEntity(action.ID)
	if entity == nil {
		return
	}

	actionKey := fmt.Sprintf("%T:%s", action, entity.ID().String())
	if !game.checkLastActionTime(actionKey, action.Created, moveThrottle) {
		return
	}
	position := entity.Position()
	pos := action.Position
	if d := position.Distance(pos); d > maxMove {
		pos.Y /= d
		pos.X /= d
	}
	entity.Set(pos)
	// Inform the client that the entity moved.
	change := MoveChange{
		Entity:   entity,
		Position: pos,
	}
	game.sendChange(change)
	game.updateLastActionTime(actionKey, action.Created)
}
