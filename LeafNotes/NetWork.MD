# Leaf的NetWork模块浅析

network目录下

分为tcp连接、websocket连接、json消息格式、protobuf消息格式

json和protobuf作为一个消息上下文处理器，提供消息注册，设置路由，消息
的序列化和反序列化。可以在这边对序列化和反序列化进行修改，进行
消息的加密

processor.go定义一个消息上下文处理器所必须实现的三个接口
```golang
type Processor interface {
	// must goroutine safe
	//路由
	Route(msg interface{}, userData interface{}) error
	// must goroutine safe
	//反序列化
	Unmarshal(data []byte) (interface{}, error)
	// must goroutine safe
	//序列化
	Marshal(msg interface{}) ([][]byte, error)
}
```

还有另外两个方法接口分别是conn.go：
```golang
type Conn interface {
    //读取消息
	ReadMsg() ([]byte, error)
    //发送消息
	WriteMsg(args ...[]byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
}
```
和agent：
```golang
type Agent interface {
    //运行
	Run()
	//断开连接时
	OnClose()
}
```

这里只分析websocket，tcp类似：
* ws_client.go websocket的客户端
* ws_conn.go 处理连接
* ws_server.go 作为websocket的服务端，长监听

#### ws_server.go
```golang
type WSServer struct {
    //监听地址
	Addr            string
	//最大连接数量
	MaxConnNum      int
	PendingWriteNum int
	//单次最大消息长度
	MaxMsgLen       uint32
	//Http超时时间
	HTTPTimeout     time.Duration
	CertFile        string
	KeyFile         string
	//返回一个Agent接口
	NewAgent        func(*WSConn) Agent
	//监听器
	ln              net.Listener
	//当收到消息是，监听器会调用该函数来处理消息
	handler         *WSHandler
}
//websocket处理程序
type WSHandler struct {
	maxConnNum      int
	pendingWriteNum int
	maxMsgLen       uint32
	newAgent        func(*WSConn) Agent
	upgrader        websocket.Upgrader
	conns           WebsocketConnSet
	mutexConns      sync.Mutex
	wg              sync.WaitGroup
}
```
一般gate作为一个moudle模块接口，处理客户端的连接，进行消息的转发，消息的下发。
所以一般network会作为gate的结构，在gate中进行初始化。gate模块`OnInit()`,
`OnDestry()`是在使用者自己定义，它的`Run()`已经被Leaf实现了，其中的`Run`方法
进行network的初始化：
```golang
var wsServer *network.WSServer
if gate.WSAddr != "" {
	wsServer = new(network.WSServer)
	wsServer.Addr = gate.WSAddr
	wsServer.MaxConnNum = gate.MaxConnNum
	wsServer.PendingWriteNum = gate.PendingWriteNum
	wsServer.MaxMsgLen = gate.MaxMsgLen
	wsServer.HTTPTimeout = gate.HTTPTimeout
	wsServer.CertFile = gate.CertFile
	wsServer.KeyFile = gate.KeyFile
	wsServer.NewAgent = func(conn *network.WSConn) network.Agent {
		a := &agent{conn: conn, gate: gate}
		if gate.AgentChanRPC != nil {
			gate.AgentChanRPC.Go("NewAgent", a)
		}
		return a
	}
}
```

