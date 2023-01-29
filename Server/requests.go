package main

import (
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"log"
)

func (s *GameServer) handleRequests(request *pb.Request, client *client, game *backend.Game) {
	switch request.GetAction().(type) {
	case *pb.Request_MoveEntity:
		s.handleMoveRequest(game, request.GetMoveEntity())
	case *pb.Request_AddEntity:
		s.handleAddRequest(game, request.GetAddEntity(), client.id)
	case *pb.Request_RemoveEntity:
		s.handleRemoveRequest(game, request.GetRemoveEntity())
	case *pb.Request_UpdateEntity:
		s.handleRemoveRequest(game, request.GetRemoveEntity())
	default:
		log.Println("unknown request", request)
	}
}

func (s *GameServer) handleMoveRequest(game *backend.Game, req *pb.MoveEntity) {
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

func (s *GameServer) handleAddRequest(game *backend.Game, req *pb.AddEntity, clientId backend.Token) {
	ent, err := backend.EntityFromProto(req.GetEntity())
	if err != nil {
		log.Println("can't parse entity to add")
		return
	}

	game.AddEntity(ent, clientId, !req.GetKeepOnDisconnect())
}
