package main

//todo rewrite this
// empty sersions get deleted
// have server ping everyone to find disconnects // keepalive
// multiple sends in one broadcast
// dont send requests to sender
// have send ticks
// change the colour to a bytes object so arbitrary data can be stored, also players can call a function to send a update entity requst

import (
	"context"
	"errors"
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"io"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"
)

const (
	clientTimeout = 2 * time.Minute
	maxClients    = 4
)

type client struct {
	streamServer pb.Game_StreamServer
	lastMessage  time.Time
	done         chan error
	id           backend.Token
	session      *backend.Game
}

// GameServer is used to stream game information with clients.
type GameServer struct {
	pb.UnimplementedGameServer
	ChangeChannel chan backend.Change
	games         map[string]*backend.Game
	sessionUsers  map[string][]backend.Token
	clients       map[backend.Token]*client
	mu            sync.RWMutex
}

// NewGameServer constructs a new game server struct.
func NewGameServer() *GameServer {
	server := &GameServer{
		games:         make(map[string]*backend.Game),
		clients:       make(map[backend.Token]*client),
		ChangeChannel: make(chan backend.Change, 10),
		sessionUsers:  make(map[string][]backend.Token),
	}
	server.watchChanges()
	server.watchTimeout()
	return server
}

func (s *GameServer) Stop() {
	for id := range s.games {
		log.Printf("Stopping \"%s\"\n", id)
		s.removeSession(id)
	}
}

func (s *GameServer) addClient(c *client) {
	s.mu.Lock()
	s.clients[c.id] = c
	s.sessionUsers[c.session.GameId] = append(s.sessionUsers[c.session.GameId], c.id)
	s.mu.Unlock()
}

func (s *GameServer) removeClient(id backend.Token) {
	log.Printf("%s - removing client", id)

	c := s.clients[id]
	session := c.session
	session.RemoveClientsEntities(c.id)
	s.mu.Lock()
	delete(s.clients, id)
	users, ok := s.sessionUsers[session.GameId]
	if !ok {
		s.mu.Unlock()
		return
	}
	serverUsers := make([]backend.Token, len(users)-1, maxClients)
	count := 0
	for _, i := range users {
		if i == c.id {
			continue
		}
		serverUsers[count] = i
		count++
	}
	s.sessionUsers[session.GameId] = serverUsers
	s.mu.Unlock()
	if len(serverUsers) == 0 {
		s.removeSession(session.GameId)
	}
}

func (s *GameServer) makeSession(id string) {
	s.mu.Lock()
	_, ok := s.games[id]
	if !ok {
		s.games[id] = backend.NewGame(s.ChangeChannel, id)
		s.sessionUsers[id] = make([]backend.Token, 0, maxClients)
		s.games[id].Start()
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
	currentClient, ok := s.clients[backend.Token(uid)]
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
	var doneError error

	go func() {
		var req *pb.Request
		for {
			req, err = srv.Recv()
			if err == io.EOF {
				currentClient.done <- nil
				return
			}
			if err != nil {
				log.Printf("receive error %v", err)
				currentClient.done <- errors.New("failed to receive request")
				return
			}
			//log.Printf("got message %+v", req)
			currentClient.lastMessage = time.Now()
			s.handleRequests(req, currentClient, game)
		}
	}()
	select {
	case doneError = <-currentClient.done:
	case <-ctx.Done():
		doneError = ctx.Err()
	}

	if doneError != nil {
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

	token := backend.NewToken()
	s.addClient(&client{
		id:          token,
		done:        make(chan error, 1),
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

func (s *GameServer) broadcast(session string, resp *pb.Response) {
	s.mu.Lock()
	for id, currentClient := range s.clients {
		if currentClient.streamServer == nil {
			continue
		}
		if session != "" && currentClient.session.GameId != session {
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
