package wstest

import (
	"GoLeafServer/leafserver/src/server/msg"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"net/url"
	"reflect"
	"testing"
)

var addr = flag.String("addr", "localhost:13001", "http service address")

var u = url.URL{Scheme: "ws", Host: *addr, Path: "/"}

func TestServer(t *testing.T){
	wsConn := Connect(u.String())

	C2S_SendMsg(wsConn)
	for {
		msgType,p,err := wsConn.ReadMessage()
		if err != nil {
			fmt.Println(msgType,p,err)
			wsConn.Close()
			return
		}
		fmt.Println(msgType,p,err)
		wsConn.Close()
		return
	}
}

func Connect(urlString string)(*websocket.Conn){
	wsconn,_,err := websocket.DefaultDialer.Dial(urlString,nil)
	if err !=nil {
		fmt.Printf("Connect server err : %v \n" ,err)
		return nil
	}
	return wsconn
}

func C2S_SendMsg(ws *websocket.Conn){
	m := &msg.Hello{}
	m.Name = "I'm Client"
	yourMsg,err := Marshal(m)
	if err != nil {
		fmt.Println("marshal error",err)
		return
	}

	WriteMsg(ws,yourMsg...)

}
func Marshal(msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		return nil, errors.New("json message pointer required")
	}
	msgID := msgType.Elem().Name()
	m := map[string]interface{}{msgID: msg}
	data, err := json.Marshal(m)
	return [][]byte{data}, err
}

func WriteMsg(ws *websocket.Conn,args ...[]byte) error {

	// get len
	var msgLen uint32
	for i := 0; i < len(args); i++ {
		msgLen += uint32(len(args[i]))
	}

	// check len
	if msgLen > 5000 {
		return errors.New("message too long")
	} else if msgLen < 1 {
		return errors.New("message too short")
	}

	//// don't copy
	//if len(args) == 1 {
	//	wsConn.doWrite(args[0])
	//	return nil
	//}

	// merge the args
	msg := make([]byte, msgLen)
	l := 0
	for i := 0; i < len(args); i++ {
		copy(msg[l:], args[i])
		l += len(args[i])
	}

	//wsConn.doWrite(msg)
	ws.WriteMessage(websocket.BinaryMessage, msg)
	return nil
}

func SendMessage(ws *websocket.Conn, args []byte) {
	ws.WriteMessage(websocket.BinaryMessage, args)
}
