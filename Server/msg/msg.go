package msg

import (
	pb "exampleMulti/msg/proto"
	"github.com/name5566/leaf/network/protobuf"
)

var Processor = protobuf.NewProcessor()

func init() {
	Processor.Register(&pb.CurrentColour{})
	Processor.Register(&pb.NewColour{})
}
