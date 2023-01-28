package backend

import (
	pb "exampleMulti/proto"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ActionThrottle = time.Second / 80
	maxMove        = 5
)

type Game struct {
	Entities      map[uuid.UUID]*Entity
	Mu            sync.RWMutex
	ActionChannel chan Action
	ChangeChannel chan Change
	done          chan interface{}
	lastAction    map[string]time.Time
	GameId        string

	ownedEntities map[uuid.UUID][]uuid.UUID
}

// NewGame constructs a new Game struct.
func NewGame(ChangeChannel chan Change, sessionId string) *Game {
	game := Game{
		Entities:      make(map[uuid.UUID]*Entity),
		ActionChannel: make(chan Action, 1),
		ChangeChannel: ChangeChannel,
		lastAction:    make(map[string]time.Time),

		ownedEntities: make(map[uuid.UUID][]uuid.UUID),
		GameId:        sessionId,
	}
	return &game
}

func (g *Game) Start() {
	go g.watchActions()
}

func (g *Game) Stop() {
	g.done <- true
}

func (g *Game) watchActions() {
	for {
		select {
		case action := <-g.ActionChannel:
			g.Mu.Lock()
			action.Perform(g)
			g.Mu.Unlock()
		case <-g.done:
			return
		}
	}
}

func (g *Game) GetProtoEntities() []*pb.Entity {
	g.Mu.RLock()
	entities := make([]*pb.Entity, 0, len(g.Entities))
	for _, entity := range g.Entities {
		entities = append(entities, entity.ToProto())
	}
	g.Mu.RUnlock()
	return entities
}

// AddEntity adds an entity to the game.
func (g *Game) AddEntity(entity *Entity, client uuid.UUID, RemoveOnDisconnect bool) {
	g.ActionChannel <- &AddAction{
		baseAction:         g.getBaseAction(entity.ID),
		Entity:             entity,
		ClientID:           client,
		RemoveOnDisconnect: RemoveOnDisconnect,
	}
}

func (g *Game) MoveEntity(id uuid.UUID, position Coordinate) {
	g.ActionChannel <- MoveAction{
		baseAction: g.getBaseAction(id),
		Position:   position,
	}
}

// RemoveEntity removes an entity from the game.
func (g *Game) RemoveEntity(id uuid.UUID) {
	entity := g.Entities[id]
	g.ActionChannel <- &RemoveAction{
		baseAction: g.getBaseAction(id),
		Entity:     entity,
	}
}

// RemoveClientsEntities clears out all entities belonging to a user
func (g *Game) RemoveClientsEntities(id uuid.UUID) {
	entities, ok := g.ownedEntities[id]
	if !ok {
		return
	}
	g.Mu.Lock()
	for _, eid := range entities {
		g.RemoveEntity(eid)
	}
	delete(g.ownedEntities, id)
	g.Mu.Unlock()
}

func (g *Game) sendChange(change Change) {
	g.ChangeChannel <- change
}

type Event interface {
	EntityID() uuid.UUID
	GameID() string
}

type baseEvent struct {
	id  uuid.UUID
	gId string
}

func (b baseEvent) EntityID() uuid.UUID {
	return b.id
}
func (b baseEvent) GameID() string {
	return b.gId
}
