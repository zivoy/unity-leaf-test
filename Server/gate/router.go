package gate

import (
	"exampleMulti/game"
	"exampleMulti/msg"
	pb "exampleMulti/msg/proto"
)

func init() {
	msg.Processor.SetRouter(&pb.CurrentColour{}, game.ChanRPC)
	msg.Processor.SetRouter(&pb.NewColour{}, game.ChanRPC)
}
