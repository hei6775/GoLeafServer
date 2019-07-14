package main

import (
	"github.com/name5566/leaf"
	lconf "github.com/name5566/leaf/conf"

	"GoLeafServer/leafserver/src/server/game"
	"GoLeafServer/leafserver/src/server/gate"
	"GoLeafServer/leafserver/src/server/login"
	"GoLeafServer/leafserver/src/server/conf"
)

func main() {
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
