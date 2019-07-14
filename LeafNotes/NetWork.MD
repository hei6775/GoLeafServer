# Leaf的NetWork模块浅析

leaf是如何接收发送消息的？

network目录下

分为tcp连接、websocket连接、json消息格式、protobuf消息格式四个部分

json和protobuf作为一个消息上下文处理器，提供消息注册，设置路由，消息
的序列化和反序列化。可以在这边对序列化和反序列化进行修改，进行
消息的加密

processor.go定义一个消息上下文处理器，所必须实现的三个接口
```go
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
```go
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
```go
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
```go
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
```go
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
        //这里的agent是gate中定义的agent
        a := &agent{conn: conn, gate: gate}
        //向gate中关联的模块发起RPC调用
        if gate.AgentChanRPC != nil {
            gate.AgentChanRPC.Go("NewAgent", a)
        }
        return a
    }
}
//然后调用websocket的start方法
if wsServer != nil {
	wsServer.Start()
}
```
这里可以看到，在gate中做统一管理，当产生一个新的连接时，会向agentChanRPC发起调用
```go
func (server *WSServer) Start() {
    ln, err := net.Listen("tcp", server.Addr)
    if err != nil {
    log.Fatal("%v", err)
    }
    if server.MaxConnNum <= 0 {
        server.MaxConnNum = 100
        log.Release("invalid MaxConnNum, reset to %v", server.MaxConnNum)
    }
    if server.PendingWriteNum <= 0 {
        server.PendingWriteNum = 100
        log.Release("invalid PendingWriteNum, reset to %v", server.PendingWriteNum)
    }
    if server.MaxMsgLen <= 0 {
        server.MaxMsgLen = 4096
        log.Release("invalid MaxMsgLen, reset to %v", server.MaxMsgLen)
    }
    if server.HTTPTimeout <= 0 {
        server.HTTPTimeout = 10 * time.Second
        log.Release("invalid HTTPTimeout, reset to %v", server.HTTPTimeout)
    }
    if server.NewAgent == nil {
        log.Fatal("NewAgent must not be nil")
    }
    
    if server.CertFile != "" || server.KeyFile != "" {
        config := &tls.Config{}
        config.NextProtos = []string{"http/1.1"}
        
        var err error
        config.Certificates = make([]tls.Certificate, 1)
        config.Certificates[0], err = tls.LoadX509KeyPair(server.CertFile, server.KeyFile)
        if err != nil {
            log.Fatal("%v", err)
        }
        
        ln = tls.NewListener(ln, config)
    }

    server.ln = ln
    server.handler = &WSHandler{
        maxConnNum:      server.MaxConnNum,
        pendingWriteNum: server.PendingWriteNum,
        maxMsgLen:       server.MaxMsgLen,
        newAgent:        server.NewAgent,
        conns:           make(WebsocketConnSet),
        upgrader: websocket.Upgrader{
            HandshakeTimeout: server.HTTPTimeout,
            CheckOrigin:      func(_ *http.Request) bool { return true },
        },
    }

    httpServer := &http.Server{
        Addr:           server.Addr,
        Handler:        server.handler,
        ReadTimeout:    server.HTTPTimeout,
        WriteTimeout:   server.HTTPTimeout,
        MaxHeaderBytes: 1024,
    }
    go httpServer.Serve(ln)
}

```
在`start`中，首先调用了一个基于tcp的套接字监听器，然后进行一些参数的初始化，
再将一个`WSHandler`赋值给`server.handler`，建立一个`http.Server`，并将地址和
handler同样赋值给它，新起一个协程监听该连接。

`func Listen(net, laddr string) (Listener, error)`的含义：

返回在一个本地网络地址laddr上监听的Listener。网络类型参数net必须是面向流的网络：
"tcp"、"tcp4"、"tcp6"、"unix"或"unixpacket"。参见Dial函数获取laddr的语法。

`func (srv *Server) Serve(l net.Listener) error`的含义：

`Serve`会接手监听器`l`收到的每一个连接，并为每一个连接创建一个新的服务go程。
该go程会读取请求，然后调用`srv.Handler`回复请求。这边的`Handler`已经被Leaf设置为
自定义的`WSHandler`，其实`Handler`是一个方法接口：
```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```
Leaf自定义的`WSHandler`已经实现了该方法：
```go
func (handler *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//请求方法 
	// if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	//请求升级为websocket长连接
	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debug("upgrade error: %v", err)
		return
	}
	//设置每次读取的最大消息大小，如果超过该大小，连接会被关闭
	conn.SetReadLimit(int64(handler.maxMsgLen))
    
	handler.wg.Add(1)
	defer handler.wg.Done()

	handler.mutexConns.Lock()
	if handler.conns == nil {
		handler.mutexConns.Unlock()
		conn.Close()
		return
	}
	if len(handler.conns) >= handler.maxConnNum {
		handler.mutexConns.Unlock()
		conn.Close()
		log.Debug("too many connections")
		return
	}
	handler.conns[conn] = struct{}{}
	handler.mutexConns.Unlock()
	
    //调用ws_conn的newWSConn函数,生成Leaf自定义的连接WSConn
	wsConn := newWSConn(conn, handler.pendingWriteNum, handler.maxMsgLen)
	//包装，返回agent接口（Run,OnClose方法）
	agent := handler.newAgent(wsConn)
	//执行Run方法,是一个for循环
	agent.Run()
    
	//当退出for循环时调用Close方法
	// cleanup
	wsConn.Close()
	handler.mutexConns.Lock()
	delete(handler.conns, conn)
	handler.mutexConns.Unlock()
	//调用agent的关闭方法，向注册的chan中发消息
	agent.OnClose()
}
```
咱们顺着`ServeHTTP`方法，这是接受一个http请求并将该请求升级为websocket连接，并且进行了一定的参数设置
以及条件判断，直到生成Leaf中自定义的WSConn，这个WSConn主要是将websocket的conn包装进去，并设置其他字段，
例如`writeChan chan []byte`用来读取其他模块的通道发过来的要下发的消息，并通过conn发送到客户端，当然为了
不阻塞当前协程，是利用关键字go另起一个协程来做下发的。
如下代码：
```go
func newWSConn(conn *websocket.Conn, pendingWriteNum int, maxMsgLen uint32) *WSConn {
	wsConn := new(WSConn)
	wsConn.conn = conn
	wsConn.writeChan = make(chan []byte, pendingWriteNum)
	wsConn.maxMsgLen = maxMsgLen

	go func() {
		for b := range wsConn.writeChan {
			if b == nil {
				break
			}

			err := conn.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				break
			}
		}

		conn.Close()
		wsConn.Lock()
		wsConn.closeFlag = true
		wsConn.Unlock()
	}()

	return wsConn
}
```
这个时候gate中定义的生成agent的函数就有用了，包装conn并返回一个agent，随后调用`agent.Run()`方法：
```go
func (a *agent) Run() {
	for {
		data, err := a.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}

		if a.gate.Processor != nil {
			msg, err := a.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			err = a.gate.Processor.Route(msg, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}
```
可以看到`Run`方法是一个for循环，不停的读取消息，进行反序列化，并调用Route方法。至此可以知道，
每个来自客户端的连接都会起两个新的goroutine来保证消息的接收和发送，如果客户端断开连接，则会调用
`agent`的`OnClose()`方法（在gate目录下）。
```go
func (a *agent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		err := a.gate.AgentChanRPC.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
}
```
可以看到这边是会发起CloseAgent的调用。一般这里是发送到game模块中。
