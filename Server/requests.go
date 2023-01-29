package main

import (
	"exampleMulti/backend"
	pb "exampleMulti/proto"
	"log"
)

func (s *GameServer) handleRequests(request *pb.Request, client *backend.Client, game *backend.Game) {
	switch request.GetAction().(type) {
	case *pb.Request_MoveEntity:
		s.handleMoveRequest(game, client, request.GetMoveEntity())
	case *pb.Request_AddEntity:
		s.handleAddRequest(game, client, request.GetAddEntity())
	case *pb.Request_RemoveEntity:
		s.handleRemoveRequest(game, client, request.GetRemoveEntity())
	case *pb.Request_UpdateEntity:
		s.handleUpdateRequest(game, client, request.GetUpdateEntity())
	default:
		log.Println("unknown request", request)
	}
}

func (s *GameServer) handleMoveRequest(game *backend.Game, client *backend.Client, req *pb.MoveEntity) {
	id, ok := ParseUUID(req.GetId())
	if !ok {
		log.Println("can't parse id to move")
		return
	}
	game.MoveEntity(id, client, backend.CoordinateFromProto(req.GetPosition()))
}

func (s *GameServer) handleRemoveRequest(game *backend.Game, client *backend.Client, req *pb.RemoveEntity) {
	id, ok := ParseUUID(req.GetId())
	if !ok {
		log.Println("can't parse id to remove")
		return
	}
	game.RemoveEntity(id, client)
}

func (s *GameServer) handleAddRequest(game *backend.Game, client *backend.Client, req *pb.AddEntity) {
	ent, err := backend.EntityFromProto(req.GetEntity())
	if err != nil {
		log.Println("can't parse entity to add")
		return
	}

	game.AddEntity(ent, client, !req.GetKeepOnDisconnect())
}

func (s *GameServer) handleUpdateRequest(game *backend.Game, client *backend.Client, req *pb.UpdateEntity) {
	ent, err := backend.EntityFromProto(req.GetEntity())
	if err != nil {
		log.Println("can't parse entity to update")
		return
	}

	game.UpdateEntity(ent, client)
}
