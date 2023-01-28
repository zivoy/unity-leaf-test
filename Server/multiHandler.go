package main

//todo rewrite this
// there will be a call that lists active sessions
// you can make a new session by submiting a key with connect
// empty sersions get deleted
import (
	"context"
	"errors"
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"image/color"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const (
	clientTimeout = 2 * time.Minute
	maxClients    = 4
)

// client contains information about connected clients.
type client struct {
	streamServer pb.Game_StreamServer
	lastMessage  time.Time
	done         chan error
	objects      []uuid.UUID
	id           uuid.UUID
}

// GameServer is used to stream game information with clients.
type GameServer struct {
	pb.UnimplementedGameServer
	game    *backend.Game
	clients map[uuid.UUID]*client
	mu      sync.RWMutex
}

// NewGameServer constructs a new game server struct.
func NewGameServer(game *backend.Game) *GameServer {
	server := &GameServer{
		game:    game,
		clients: make(map[uuid.UUID]*client),
	}
	server.watchChanges()
	server.watchTimeout()
	return server
}

func (s *GameServer) removeClient(id uuid.UUID) {
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
}

func (s *GameServer) removeRemovables(entityIDs []uuid.UUID) {
	for _, entityID := range entityIDs {
		s.removeEntity(entityID)
	}
}

func (s *GameServer) removeEntity(entityID uuid.UUID) {
	s.game.Mu.Lock()
	s.game.RemoveEntity(entityID)
	s.game.Mu.Unlock()

	resp := pb.Response{
		Action: &pb.Response_RemoveEntity{
			RemoveEntity: &pb.RemoveEntity{
				Id: entityID.String(),
			},
		},
	}
	s.broadcast(&resp)

}

func (s *GameServer) getClientFromContext(ctx context.Context) (*client, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	tokenRaw := headers["authorization"]
	if len(tokenRaw) == 0 {
		return nil, errors.New("no token provided")
	}
	token, ok := pb.ParseUUID(tokenRaw[0])
	if !ok {
		return nil, errors.New("cannot parse token")
	}
	s.mu.RLock()
	currentClient, ok := s.clients[token]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("token not recognized")
	}
	return currentClient, nil
}

// Stream is the main loop for dealing with individual players.
func (s *GameServer) Stream(srv pb.Game_StreamServer) error {
	ctx := srv.Context()

	currentClient, err := s.getClientFromContext(ctx)
	if err != nil {
		return err
	}
	if currentClient.streamServer != nil {
		return errors.New("stream already active")
	}
	currentClient.streamServer = srv

	log.Println("start new server")

	// Wait for stream requests.
	go func() {
		for {
			req, err := srv.Recv()
			if err != nil {
				log.Printf("receive error %v", err)
				currentClient.done <- errors.New("failed to receive request")
				return
			}
			// log.Printf("got message %+v", req)
			currentClient.lastMessage = time.Now()
			switch req.GetAction().(type) {
			case *pb.Request_Move:
				s.handleMoveRequest(req)
			case *pb.Request_AddEntity:
				s.handleAddRequest(req, currentClient)
			case *pb.Request_RemoveEntity:
				s.handleRemoveRequest(req)
			}
		}
	}()

	// Wait for stream to be done.
	var doneError error
	select {
	case <-ctx.Done():
		doneError = ctx.Err()
	case doneError = <-currentClient.done:
	}
	log.Printf(`stream done with error "%v"`, doneError)

	log.Printf("%s - removing client", currentClient.id)
	s.removeClient(currentClient.id)
	s.removeRemovables(currentClient.objects)

	return doneError
}

func (s *GameServer) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	if len(s.clients) >= maxClients {
		return nil, errors.New("the server is full")
	}

	if len(req.Name) > 16 {
		return nil, errors.New("invalid name provided")
	}
	log.Println("Incoming connection from", req.Name)

	colour := color.RGBA{
		R: uint8(rand.Intn(256)),
		G: uint8(rand.Intn(256)),
		B: uint8(rand.Intn(256)),
	}

	// Choose a random spawn point.
	rand.Seed(time.Now().Unix())
	startCoordinate := backend.Coordinate{Y: 0, X: 0}

	player := &backend.Entity{
		Name:            req.Name,
		Colour:          colour,
		IdentifierBase:  backend.IdentifierBase{UUID: uuid.New()}, //todo why is the user being regiestered here
		CurrentPosition: startCoordinate,
	}
	s.game.Mu.Lock()
	s.game.AddEntity(player)
	s.game.Mu.Unlock()

	// Build a slice of current entities.
	s.game.Mu.RLock()
	entities := make([]*pb.Entity, 0, len(s.game.Entities))
	for _, entity := range s.game.Entities {
		pbEntity := pb.GetProtoEntity(entity)
		if pbEntity != nil {
			entities = append(entities, pbEntity)
		}
	}
	s.game.Mu.RUnlock()

	// Add the new client.
	s.mu.Lock()
	token := uuid.New()
	s.clients[token] = &client{
		id:          token,
		objects:     make([]uuid.UUID, 0),
		done:        make(chan error),
		lastMessage: time.Now(),
	}
	s.mu.Unlock()

	return &pb.ConnectResponse{
		Token:    token.String(),
		Entities: entities,
	}, nil
}

func (s *GameServer) watchTimeout() {
	timeoutTicker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			for _, client := range s.clients {
				if time.Since(client.lastMessage) > clientTimeout {
					client.done <- errors.New("you have been timed out")
					return
				}
			}
			<-timeoutTicker.C
		}
	}()
}

// WatchChanges waits for new game engine changes and broadcasts to clients.
func (s *GameServer) watchChanges() {
	go func() {
		for {
			change := <-s.game.ChangeChannel
			switch change.(type) {
			case backend.MoveChange:
				change := change.(backend.MoveChange)
				s.handleMoveChange(change)
			}
		}
	}()
}

// broadcast sends a response to all clients.
func (s *GameServer) broadcast(resp *pb.Response) {
	s.mu.Lock()
	for id, currentClient := range s.clients {
		if currentClient.streamServer == nil {
			continue
		}
		if err := currentClient.streamServer.Send(resp); err != nil {
			log.Printf("%s - broadcast error %v", id, err)
			currentClient.done <- errors.New("failed to broadcast message")
			continue
		}
		//log.Printf("%s - broadcasted %+v", resp, id)
	}
	s.mu.Unlock()
}

// handleMoveRequest makes a request to the game engine to move a player.
func (s *GameServer) handleMoveRequest(req *pb.Request) {
	move := req.GetMove()
	id, ok := pb.ParseUUID(move.GetId())
	if !ok {
		log.Println("can't parse id to move")
		return
	}
	s.game.ActionChannel <- backend.MoveAction{
		ID:       id,
		Position: pb.GetBackendCoordinate(move.GetPosition()),
		Created:  time.Now(),
	}
}

func (s *GameServer) handleRemoveRequest(req *pb.Request) {
	remove := req.GetRemoveEntity()
	if id, ok := pb.ParseUUID(remove.Id); ok {
		s.removeEntity(id)
		return
	}
	log.Println("can't parse id to remove")
}

func (s *GameServer) handleAddRequest(req *pb.Request, activeClient *client) {
	add := req.GetAddEntity()
	ent := add.GetEntity()

	if id, ok := pb.ParseUUID(ent.GetId()); !add.GetKeepOnDisconnect() && ok {
		activeClient.objects = append(activeClient.objects, id)
	}

	resp := pb.Response{
		Action: &pb.Response_AddEntity{
			AddEntity: &pb.AddEntity{
				Entity: ent,
			},
		},
	}
	s.broadcast(&resp)
}

func (s *GameServer) handleMoveChange(change backend.MoveChange) {
	resp := pb.Response{
		Action: &pb.Response_UpdateEntity{
			UpdateEntity: &pb.UpdateEntity{
				Entity: pb.GetProtoEntity(change.Entity),
			},
		},
	}
	s.broadcast(&resp)
}
