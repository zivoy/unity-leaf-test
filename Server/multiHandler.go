package main

import (
	"context"
	"errors"
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"image/color"
	"log"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const (
	clientTimeout = 2 * time.Minute
	maxClients    = 8
)

// client contains information about connected clients.
type client struct {
	streamServer pb.Game_StreamServer
	lastMessage  time.Time
	done         chan error
	playerID     uuid.UUID
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

func (s *GameServer) removePlayer(playerID uuid.UUID) {
	s.game.Mu.Lock()
	s.game.RemoveEntity(playerID)
	s.game.Mu.Unlock()

	resp := pb.Response{
		Action: &pb.Response_RemoveEntity{
			RemoveEntity: &pb.RemoveEntity{
				Id: playerID.String(),
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
	token, err := uuid.Parse(tokenRaw[0])
	if err != nil {
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
				s.handleMoveRequest(req, currentClient)
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
	s.removePlayer(currentClient.playerID)

	return doneError
}

func (s *GameServer) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	if len(s.clients) >= maxClients {
		return nil, errors.New("the server is full")
	}

	re := regexp.MustCompile("^[a-zA-Z0-9 _-]+$")
	if !re.MatchString(req.Name) {
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

	// make up an id for the user
	id := uuid.New()

	// Add the player.
	player := &backend.Player{
		Name:            req.Name,
		Colour:          colour,
		IdentifierBase:  backend.IdentifierBase{UUID: id},
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

	// Inform all other clients of the new player.
	resp := pb.Response{
		Action: &pb.Response_AddEntity{
			AddEntity: &pb.AddEntity{
				Entity: pb.GetProtoEntity(player),
			},
		},
	}
	s.broadcast(&resp)

	// Add the new client.
	s.mu.Lock()
	token := uuid.New()
	s.clients[token] = &client{
		id:          token,
		playerID:    id,
		done:        make(chan error),
		lastMessage: time.Now(),
	}
	s.mu.Unlock()

	return &pb.ConnectResponse{
		Id:       id.String(),
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
			case backend.AddEntityChange:
				change := change.(backend.AddEntityChange)
				s.handleAddEntityChange(change)
			case backend.RemoveEntityChange:
				change := change.(backend.RemoveEntityChange)
				s.handleRemoveEntityChange(change)
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
func (s *GameServer) handleMoveRequest(req *pb.Request, currentClient *client) {
	move := req.GetMove()
	s.game.ActionChannel <- backend.MoveAction{
		ID:       currentClient.playerID,
		Position: pb.GetBackendCoordinate(move),
		Created:  time.Now(),
	}
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

func (s *GameServer) handleAddEntityChange(change backend.AddEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_AddEntity{
			AddEntity: &pb.AddEntity{
				Entity: pb.GetProtoEntity(change.Entity),
			},
		},
	}
	s.broadcast(&resp)
}

func (s *GameServer) handleRemoveEntityChange(change backend.RemoveEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_RemoveEntity{
			RemoveEntity: &pb.RemoveEntity{
				Id: change.Entity.ID().String(),
			},
		},
	}
	s.broadcast(&resp)
}
