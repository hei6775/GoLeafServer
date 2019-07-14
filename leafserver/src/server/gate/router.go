package gate

import (
	"GoLeafServer/leafserver/src/server/game"
	"GoLeafServer/leafserver/src/server/msg"
)

func init() {
	msg.Processor.SetRouter(&msg.Hello{},game.ChanRPC)
}
