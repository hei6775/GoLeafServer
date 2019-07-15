# Leaf的ChanRPC通信结构

## 简介
>ChanRPC 的调用方有 3 种调用模式：
> 
> 同步模式，调用并等待 ChanRPC 返回
> 异步模式，调用并提供回调函数，回调函数会在 ChanRPC 返回后被调用
> Go 模式，调用并立即返回，忽略任何返回值和错误
>

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

//初始化
func NewClient(l int) *Client {
	c := new(Client)
	c.chanSyncRet = make(chan *RetInfo, 1)
	c.ChanAsynRet = make(chan *RetInfo, l)
	return c
}

func (c *Client) Attach(s *Server) {
	c.s = s
}
```
对于通道的缓冲大小设置在LeafServer中默认设置就是：
```go
// skeleton conf
GoLen              = 10000 //go机制的回调函数通道缓冲大小
TimerDispatcherLen = 10000 //leaf的定时器机制的定时器通道缓冲大小
AsynCallLen        = 10000 //RPC client的异步结果通知通道缓冲大小
ChanRPCLen         = 10000 //RPC server的CallInfo的通道缓冲大小
```
但是如果没有设置这些参数，而是默认的0呢？会发生什么情况？回到`client`，它有几个方法：
```go
func (c *Client) call(ci *CallInfo, block bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if block {
		c.s.ChanCall <- ci
	} else {
		select {
		case c.s.ChanCall <- ci:
		default:
			err = errors.New("chanrpc channel full")
		}
	}
	return
}
```
接受一个`CallInfo`以及是否是阻塞状态的标志位`block`，而这个`c.s.ChanCall`中的`ChanRet`
根据调用的方法的不同，分别采用`chanSyncRet`和`chanAsynRet`，而`chanSyncRet`是一个缓冲
大小为1的，缓冲大小为1的通道是具有延迟保障的功能，保证前一个消息已被接受。而`chanAsynRet`
被设置为10000的缓冲量的通道，不具有保证功能，但是可以保证不被阻塞。

基于这个调用方法，Leaf提供了更多的同步调用方法，如下：
```go
//f作为检测传入的id对应的函数是否存在，且是不是三种类型
func (c *Client) f(id interface{}, n int) (f interface{}, err error) {
	if c.s == nil {
		err = errors.New("server not attached")
		return
	}

	f = c.s.functions[id]
	if f == nil {
		err = fmt.Errorf("function id %v: function not registered", id)
		return
	}

	var ok bool
	switch n {
	case 0:
		_, ok = f.(func([]interface{}))
	case 1:
		_, ok = f.(func([]interface{}) interface{})
	case 2:
		_, ok = f.(func([]interface{}) []interface{})
	default:
		panic("bug")
	}

	if !ok {
		err = fmt.Errorf("function id %v: return type mismatch", id)
	}
	return
}
//调用0的类型的函数，并等待结果返回
func (c *Client) Call0(id interface{}, args ...interface{}) error {
	f, err := c.f(id, 0)
	if err != nil {
		return err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return err
	}

	ri := <-c.chanSyncRet
	return ri.err
}
//调用1类型的函数，并等待结果返回
func (c *Client) Call1(id interface{}, args ...interface{}) (interface{}, error) {
	f, err := c.f(id, 1)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet
	return ri.ret, ri.err
}
//调用2类型的函数，并等待结果返回
func (c *Client) CallN(id interface{}, args ...interface{}) ([]interface{}, error) {
	f, err := c.f(id, 2)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet
	return assert(ri.ret), ri.err
}
```
它还有对应于异步的RPC调用，由于`ChanAsynRet`是一个带有缓冲的channel所以不会阻塞当前线程，具体可看[实例代码](https://github.com/hei6775/GoLeafServer/blob/master/LeafNotes/chanrpc/chanrpc_test.go)：
```go
func (c *Client) asynCall(id interface{}, args []interface{}, cb interface{}, n int) {
	f, err := c.f(id, n)
	if err != nil {
		c.ChanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}
    //异步调用
	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.ChanAsynRet,
		cb:      cb,
	}, false)
	if err != nil {
		c.ChanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}
}

func (c *Client) AsynCall(id interface{}, _args ...interface{}) {
	if len(_args) < 1 {
		panic("callback function not found")
	}

	args := _args[:len(_args)-1]
	//回调函数默认放在参数args的最后一个
	cb := _args[len(_args)-1]

	var n int
	switch cb.(type) {
	case func(error):
		n = 0
	case func(interface{}, error):
		n = 1
	case func([]interface{}, error):
		n = 2
	default:
		panic("definition of callback function is invalid")
	}

	// too many calls
	if c.pendingAsynCall >= cap(c.ChanAsynRet) {
		execCb(&RetInfo{err: errors.New("too many calls"), cb: cb})
		return
	}

	c.asynCall(id, args, cb, n)
	c.pendingAsynCall++
}
```



