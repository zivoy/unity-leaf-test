package backend

import (
	pb "exampleMulti/proto"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Token uuid.UUID

func (t Token) UUID() uuid.UUID {
	return uuid.UUID(t)
}
func (t Token) String() string {
	return t.UUID().String()
}

func NewToken() Token {
	return Token(uuid.New())
}

type Game struct {
	Entities      map[uuid.UUID]*Entity
	Mu            sync.RWMutex
	ActionChannel chan Action
	ChangeChannel chan Change
	done          chan interface{}
	lastAction    map[string]time.Time
	GameId        string

	ownedEntities map[Token][]uuid.UUID
}

// NewGame constructs a new Game struct.
func NewGame(ChangeChannel chan Change, sessionId string) *Game {
	game := Game{
		Entities:      make(map[uuid.UUID]*Entity),
		ActionChannel: make(chan Action, 10),
		ChangeChannel: ChangeChannel,
		lastAction:    make(map[string]time.Time),
		done:          make(chan interface{}),

		ownedEntities: make(map[Token][]uuid.UUID),
		GameId:        sessionId,
	}
	return &game
}

func (g *Game) Start() {
	go g.watchActions()
}

func (g *Game) Stop() {
	close(g.done)
}

func (g *Game) watchActions() {
	for {
		select {
		case action := <-g.ActionChannel:
			g.Mu.Lock()
			g.ChangeChannel <- action.Perform(g)
			g.Mu.Unlock()
		case <-g.done:
			return
		}
	}
}

// AddEntity adds an entity to the game.
func (g *Game) AddEntity(entity *Entity, client Token, RemoveOnDisconnect bool) {
	g.ActionChannel <- &AddAction{
		baseAction:         g.getBaseAction(entity.ID),
		Entity:             entity,
		ClientID:           client,
		RemoveOnDisconnect: RemoveOnDisconnect,
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

func (g *Game) MoveEntity(id uuid.UUID, position Coordinate) {
	g.ActionChannel <- &MoveAction{
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
func (g *Game) RemoveClientsEntities(id Token) {
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
