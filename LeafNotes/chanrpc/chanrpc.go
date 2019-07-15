package chanrpc

import (
	"fmt"

	"errors"

	"github.com/name5566/leaf/log"
)

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

//客户端
type Client struct {
	s               *Server       //服务
	chanSyncRet     chan *RetInfo //同步通道
	chanAsynRet     chan *RetInfo //异步通道
	pendingAsynCall int           //异步数量
}

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

//new client
func NewClient(l int) *Client {
	c := new(Client)
	//同步
	c.chanSyncRet = make(chan *RetInfo, 1)
	//异步
	c.chanAsynRet = make(chan *RetInfo, l)
	return c
}

//client attach server
func (c *Client) Attach(s *Server) {
	c.s = s
}

//call
func (c *Client) call(ci *CallInfo, block bool) (err error) {
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

//check function interface
//and run th func
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
		err = fmt.Errorf("function id %v:return type mismatch", id)
	}
	return
}

//call 0
func (c *Client) Call0(id interface{}, args ...interface{}) error {
	//call the server's map[id]f
	//and interface change to the one of three kind functions according to the para a
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

//call 1
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

//call n
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

//异步执行
func (c *Client) asynCall(id interface{}, args []interface{}, cb interface{}, n int) {
	f, err := c.f(id, n)

	if err != nil {
		c.chanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanAsynRet,
		cb:      cb,
	}, false)

	if err != nil {
		c.chanAsynRet <- &RetInfo{err: err, cb: cb}
		return
	}
}

//异步执行
func (c *Client) AsynCall(id interface{}, _args ...interface{}) {
	if len(_args) < 1 {
		panic("callback function not found")
	}

	args := _args[:len(_args)-1]
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

	if c.pendingAsynCall >= cap(c.chanAsynRet) {
		execCb(&RetInfo{err: errors.New("too many calls"), cb: cb})
		return
	}

	c.asynCall(id, args, cb, n)
	c.pendingAsynCall++
}

func execCb(ri *RetInfo) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("%v", r)
		}
	}()
	//execute
	switch ri.cb.(type) {
	case func(error):
		ri.cb.(func(error))(ri.err)
	case func(interface{}, error):
		ri.cb.(func(interface{}, error))(ri.ret, ri.err)
	case func([]interface{}, error):
		ri.cb.(func([]interface{}, error))(assert(ri.ret), ri.err)
	default:
		panic("bug")
	}
	return
}

func (c *Client) Cb(ri *RetInfo) {
	c.pendingAsynCall--
	execCb(ri)
}

func (c *Client) Close() {
	for c.pendingAsynCall > 0 {
		c.Cb(<-c.chanAsynRet)
	}
}

func (c *Client) Idle() bool {
	return c.pendingAsynCall == 0
}
