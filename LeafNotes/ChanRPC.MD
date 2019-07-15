# Leaf的ChanRPC通信结构

## 简介
Leaf中各个模块运行在不同的`goroutine`中，模块间通过Leaf设计的一套基于`channel`轻量级的RPC
来通讯，这也是leaf中最重要的一部分。

`ChanRPC`分为`Server`和`Client`，`Server`和`Client`分别在不同的协程中，先看`Server`的结构

```go
//one server per goroutine (goroutine not safe)
type Server struct {
	functions map[interface{}]interface{} // 存储func
	ChanCall  chan *CallInfo              //通道回调
}

type CallInfo struct {
	f       interface{}   //函数
	args    []interface{} //参数
	chanRet chan *RetInfo //通道结果
	cb      interface{}   //回调函数
}

type RetInfo struct {
	ret interface{} //结果
	err error       //错误
	//callback:
	cb interface{} //回调函数
}
```
再看`Server`所携带的方法：
```go
//生成一个Server
func NewServer(l int) *Server {
	s := new(Server)

	s.functions = make(map[interface{}]interface{})
	//
	s.ChanCall = make(chan *CallInfo, l)
	return s
}

//interface to []interface
func assert(i interface{}) []interface{} {
	if i == nil {
		return nil
	} else {
		return i.([]interface{})
	}
}

//you must call the function before calling Open and Go
func (s *Server) Register(id interface{}, f interface{}) {
	switch f.(type) {
	case func([]interface{}):
	case func([]interface{}) interface{}:
	case func([]interface{}) []interface{}:
	default:
		panic(fmt.Sprintf("function id %v: definition of function is invalid", id))
	}
    //避免相同ID被重复命名，直接在运行前panic
	if _, ok := s.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registerd", id))
	}
    
	s.functions[id] = f
}

//send ret
func (s *Server) ret(ci *CallInfo, ri *RetInfo) (err error) {

	if ci.chanRet == nil {
		return
	}
	//panic转成error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	ri.cb = ci.cb
	//这边把结果通过通道传到chanSyncRet
	ci.chanRet <- ri
	return
}

func (s *Server) exec(ci *CallInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
			s.ret(ci, &RetInfo{err: fmt.Errorf("%v", r)})
		}

	}()

	//execute
	switch ci.f.(type) {
	case func([]interface{}):
		ci.f.(func([]interface{}))(ci.args)
		return s.ret(ci, &RetInfo{})
	case func([]interface{}) interface{}:
		ret := ci.f.(func([]interface{}) interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	case func([]interface{}) []interface{}:
		ret := ci.f.(func([]interface{}) []interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	}

	panic("bug")
}

//执行
func (s *Server) Exec(ci *CallInfo) {
	err := s.exec(ci)
	if err != nil {
		log.Error("%v", err)
	}
}

//goroutine safe
func (s *Server) Go(id interface{}, args ...interface{}) {
	f := s.functions[id]

	if f == nil {
		return
	}

	defer func() {
		recover()
	}()

	s.ChanCall <- &CallInfo{
		f:    f,
		args: args,
	}
}

//close server 通道
func (s *Server) Close() {
	close(s.ChanCall)

	for ci := range s.ChanCall {
		s.ret(ci, &RetInfo{
			err: errors.New("chanrpc server closed"),
		})
	}
}

//goroutine safe
//sever open return client
func (s *Server) Open(l int) *Client {
	c := NewClient(l)
	c.Attach(s)
	return c
}
```
在`skeleton`中接收`Client`发过来的`CallInfo`，调用`(s *Server) Exec(ci *CallInfo)`的函数，
对消息进行执行，但是在调用`Open`和`Go`方法之前需要提前进行注册，一般在项目中，都在`init`中
进行RPC的注册。

且Leaf的函数方法支持三种：
* `func([]interface{})`
* `func([]interface{}) interface{}`
* `func([]interface{}) []interface{}`

运行顺序基本上是，调用`Exec`传入一个被调用信息`CallInfo`，在执行`(s *Server) exec(ci *CallInfo) (err error)`
中通过执行`CallInfo`中的`f`函数，执行完成后将执行结果包装到一个新的`RetInfo`中和`CallInfo`传入
到`(s *Server) ret(ci *CallInfo, ri *RetInfo) (err error)`，在`ret`中将`CallInfo`中的
回调函数`cb`传给`RetInfo`，再将`RetInfo`传给`CallInfo`的结果通道，由`client`进行处理。

在`Close`中，通过关闭通道，并且对通道内现存的数据不进行调用，直接传结果`通道已经关闭`回去。

那么为什么`(s *Server) Open(l int) *Client`是并发安全的呢？

对于客户端`Client`而言，它的结构较为简单，基本上就是绑定一个RPC的`Server`，设置两个通道来接受调用结果：
```go
type Client struct {
	s               *Server       //服务
	chanSyncRet     chan *RetInfo //同步结果通道
	chanAsynRet     chan *RetInfo //异步结果通道
	pendingAsynCall int           //异步进行的数量
}
```


