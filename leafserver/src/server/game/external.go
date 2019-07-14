package game

import (
	"GoLeafServer/leafserver/src/server/game/internal"
)

var (
	// 实例化 game 模块
	Module  = new(internal.Module)
	// 暴露 ChanRPC
	ChanRPC = internal.ChanRPC
)
