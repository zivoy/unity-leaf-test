package main

import (
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"log"
)

func (s *GameServer) watchChanges() {
	go func() {
		for {
			change := <-s.ChangeChannel
			switch c := change.(type) {
			case *backend.UpdateEntityChange:
				s.handleUpdateChange(*c)
			case *backend.RemoveEntityChange:
				s.handleRemoveChange(*c)
			case *backend.AddEntityChange:
				s.handleAddChange(*c)
			case *backend.MoveChange:
				s.handleMoveChange(*c)
			default:
				log.Println("unknown type", c)
			}
		}
	}()
}

func (s *GameServer) handleUpdateChange(change backend.UpdateEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_UpdateEntity{
			UpdateEntity: &pb.UpdateEntity{
				Entity: change.Entity.ToProto(),
			},
		},
	}
	s.broadcast(change.Client(), &resp)
}

func (s *GameServer) handleMoveChange(change backend.MoveChange) {
	resp := pb.Response{
		Action: &pb.Response_MoveEntity{
			MoveEntity: &pb.MoveEntity{
				Position: change.Position.ToProto(),
				Id:       change.EntityID().String(),
			},
		},
	}
	s.broadcast(change.Client(), &resp)
}

func (s *GameServer) handleRemoveChange(change backend.RemoveEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_RemoveEntity{
			RemoveEntity: &pb.RemoveEntity{
				Id: change.EntityID().String(),
			},
		},
	}
	s.broadcast(change.Client(), &resp)
}

func (s *GameServer) handleAddChange(change backend.AddEntityChange) {
	resp := pb.Response{
		Action: &pb.Response_AddEntity{
			AddEntity: &pb.AddEntity{
				Entity: change.Entity.ToProto(),
			},
		},
	}
	s.broadcast(change.Client(), &resp)
}
