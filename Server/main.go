package main

import (
	"exampleMulti/conf"
	"exampleMulti/game"
	"exampleMulti/gate"
	"exampleMulti/login"
	"flag"
	"github.com/name5566/leaf"
	lconf "github.com/name5566/leaf/conf"
)

var (
	consolePort = flag.Int("c", 0, "port that the console should use")
)

func main() {
	conf.Server.ConsolePort = *consolePort

	lconf.LogLevel = conf.Server.LogLevel
	lconf.LogPath = conf.Server.LogPath
	lconf.LogFlag = conf.LogFlag
	lconf.ConsolePort = conf.Server.ConsolePort
	lconf.ProfilePath = conf.Server.ProfilePath

	leaf.Run(
		game.Module,
		gate.Module,
		login.Module,
	)
}
