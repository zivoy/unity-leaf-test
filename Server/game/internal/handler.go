package internal

import (
	"github.com/name5566/leaf/gate"
	"github.com/name5566/leaf/log"
	"reflect"
	"server/msg"
)

func init() {
	// Register the handler of `Hello` message to `game` module handleHello
	handler(&msg.n{}, handleHello)
}

func handler(m interface{}, h interface{}) {
	skeleton.RegisterChanRPC(reflect.TypeOf(m), h)
}

func handleHello(args []interface{}) {
	// Send "Hello"
	m := args[0].(*msg.Hello)
	// The receiver
	a := args[1].(gate.Agent)

	// The content of the message
	log.Debug("hello %v", m.Name)

	// Reply with a `Hello`
	a.WriteMsg(&msg.Hello{
		Name: "client",
	})
}
