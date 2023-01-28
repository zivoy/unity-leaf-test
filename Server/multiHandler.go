package main

//todo rewrite this
// empty sersions get deleted
import (
	"context"
	"errors"
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"io"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const (
	clientTimeout = 2 * time.Minute
	maxClients    = 4
)

type token uuid.UUID

func (t token) UUID() uuid.UUID {
	return uuid.UUID(t)
}
func (t token) String() string {
	return t.UUID().String()
}

type client struct {
	streamServer pb.Game_StreamServer
	lastMessage  time.Time
	done         chan error
	id           token
	session      *backend.Game
}

// GameServer is used to stream game information with clients.
type GameServer struct {
	pb.UnimplementedGameServer
	ChangeChannel chan backend.Change
	games         map[string]*backend.Game
	sessionUsers  map[string][]token
	clients       map[token]*client
	mu            sync.RWMutex
}

// NewGameServer constructs a new game server struct.
func NewGameServer() *GameServer {
	server := &GameServer{
		games:         make(map[string]*backend.Game),
		clients:       make(map[token]*client),
		ChangeChannel: make(chan backend.Change, 10),
		sessionUsers:  make(map[string][]token),
	}
	server.watchChanges()
	server.watchTimeout()
	return server
}

func (s *GameServer) addClient(c *client) {
	s.mu.Lock()
	s.clients[c.id] = c
	s.sessionUsers[c.session.GameId] = append(s.sessionUsers[c.session.GameId], c.id)
	s.mu.Unlock()
}

func (s *GameServer) removeClient(id token) {
	log.Printf("%s - removing client", id)

	c := s.clients[id]
	session := c.session
	session.RemoveClientsEntities(c.id.UUID())
	s.mu.Lock()
	delete(s.clients, id)
	users := s.sessionUsers[session.GameId]
	serverUsers := make([]token, len(users)-1, maxClients)
	count := 0
	for _, i := range users {
		if i == c.id {
			continue
		}
		serverUsers[count] = i
		count++
	}
	s.sessionUsers[session.GameId] = serverUsers
	if len(serverUsers) == 0 {
		s.removeSession(session.GameId)
	}
	s.mu.Unlock()
}

func (s *GameServer) makeSession(id string) {
	s.mu.Lock()
	_, ok := s.games[id]
	if !ok {
		s.games[id] = backend.NewGame(s.ChangeChannel, id)
		s.sessionUsers[id] = make([]token, 0, maxClients)
		log.Println("starting new session", id)
	}
	s.mu.Unlock()
}

func (s *GameServer) removeSession(id string) {
	session := s.games[id]
	session.Stop()
	for _, c := range s.sessionUsers[id] {
		s.clients[c].done <- errors.New("session has shut down")
	}
	s.mu.Lock()
	delete(s.games, id)
	delete(s.sessionUsers, id)
	s.mu.Unlock()
	log.Println("closing session", id)
}

func (s *GameServer) getClientFromContext(ctx context.Context) (*client, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	tokenRaw := headers["authorization"]
	if len(tokenRaw) == 0 {
		return nil, errors.New("no token provided")
	}
	uid, ok := ParseUUID(tokenRaw[0])
	if !ok {
		return nil, errors.New("cannot parse token")
	}
	s.mu.RLock()
	currentClient, ok := s.clients[token(uid)]
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
	game := currentClient.session

	defer s.removeClient(currentClient.id)
	// Wait for stream requests.
	go func() {
		for {
			req, err := srv.Recv()
			if err == io.EOF {
				currentClient.done <- err
				return
			} //todo check if done was closed
			if err != nil {
				log.Printf("receive error %v", err)
				currentClient.done <- errors.New("failed to receive request")
				return
			}
			// log.Printf("got message %+v", req)
			currentClient.lastMessage = time.Now()

			switch req.GetAction().(type) {
			case *pb.Request_Move:
				s.handleMoveRequest(game, req.GetMove())
			case *pb.Request_AddEntity:
				s.handleAddRequest(game, req.GetAddEntity(), uuid.UUID(currentClient.id))
			case *pb.Request_RemoveEntity:
				s.handleRemoveRequest(game, req.GetRemoveEntity())
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
	close(currentClient.done)

	if err != io.EOF {
		log.Printf(`stream done with error "%v"`, doneError)
	}

	return doneError
}

func (s *GameServer) List(ctx context.Context, req *pb.SessionRequest) (*pb.SessionList, error) {
	servers := make([]*pb.Server, 0, len(s.games))
	for u := range s.games {
		servers = append(servers, &pb.Server{
			Id:     u,
			Online: uint32(len(s.sessionUsers[u])),
			Max:    maxClients,
		})
	}
	return &pb.SessionList{
		Servers: servers,
	}, nil
}

func (s *GameServer) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	sessionId := req.GetSession()
	if len(sessionId) < 3 || len(sessionId) > 25 {
		return nil, errors.New("invalid sessionId Provided")
	}
	log.Println("Incoming connection to", req.Session)

	s.makeSession(sessionId) //todo kill unjoined servers
	sessionUsers := s.sessionUsers[sessionId]
	if len(sessionUsers) >= maxClients {
		return nil, errors.New("the server is full")
	}

	game := s.games[sessionId]
	entities := game.GetProtoEntities()

	token := token(uuid.New())
	s.addClient(&client{
		id:          token,
		done:        make(chan error),
		lastMessage: time.Now(),
		session:     game,
	})

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

func (s *GameServer) watchChanges() {
	go func() {
		for {
			change := <-s.ChangeChannel
			switch change.(type) {
			case backend.UpdateEntityChange:
				s.handleUpdateChange(change.(backend.UpdateEntityChange))
			case backend.RemoveEntityChange:
				s.handleRemoveChange(change.(backend.RemoveEntityChange))
			case backend.AddEntityChange:
				s.handleAddChange(change.(backend.AddEntityChange))
			}
		}
	}()
}

func (s *GameServer) handleMoveRequest(game *backend.Game, req *pb.MoveAction) {
	id, ok := ParseUUID(req.GetId())
	if !ok {
		log.Println("can't parse id to move")
		return
	}
	game.MoveEntity(id, backend.CoordinateFromProto(req.GetPosition()))
}

func (s *GameServer) handleRemoveRequest(game *backend.Game, req *pb.RemoveEntity) {
	id, ok := ParseUUID(req.GetId())
	if !ok {
		log.Println("can't parse id to remove")
		return
	}
	game.RemoveEntity(id)
}

func (s *GameServer) handleAddRequest(game *backend.Game, req *pb.AddEntity, clientId uuid.UUID) {
	ent, err := backend.EntityFromProto(req.GetEntity())
	if err != nil {
		log.Println("can't parse entity to add")
		return
	}

	game.AddEntity(ent, clientId, !req.GetKeepOnDisconnect())
}

func (s *GameServer) broadcast(session string, resp *pb.Response) {
	s.mu.Lock()
	for id, currentClient := range s.clients {
		if currentClient.streamServer == nil {
			continue
		}
		if currentClient.session.GameId != session {
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

func (s *GameServer) handleUpdateChange(change backend.UpdateEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_UpdateEntity{
			UpdateEntity: &pb.UpdateEntity{
				Entity: change.Entity.ToProto(),
			},
		},
	}
	s.broadcast(change.GameID(), &resp)
}

func (s *GameServer) handleRemoveChange(change backend.RemoveEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_RemoveEntity{
			RemoveEntity: &pb.RemoveEntity{
				Id: change.EntityID().String(),
			},
		},
	}
	s.broadcast(change.GameID(), &resp)
}

func (s *GameServer) handleAddChange(change backend.AddEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_AddEntity{
			AddEntity: &pb.AddEntity{
				Entity: change.Entity.ToProto(),
			},
		},
	}
	s.broadcast(change.GameID(), &resp)
}
