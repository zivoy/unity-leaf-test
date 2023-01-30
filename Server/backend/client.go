package backend

import (
	pb "exampleMulti/proto"
	"time"
)

type Client struct {
	StreamServer pb.Game_StreamServer
	LastMessage  time.Time
	Done         chan error
	Id           Token
	Session      *Game
}

func NewClient(game *Game) *Client {
	return &Client{
		Id:          NewToken(),
		Done:        make(chan error, 1),
		LastMessage: time.Now(),
		Session:     game,
	}
}