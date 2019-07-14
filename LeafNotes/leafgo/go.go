package g

import (
	"container/list"
	"sync"
	"runtime"
	"log"
)

//Go结构体
type Go struct {
	ChanCb chan func() //回调函数通道
	pendingGo int //等待的go
}

//线性的Go结构体
type LinearGo struct {
	f func()  //执行函数
	cb func() //回调函数
}

//线性的上下文
type LinearContext struct {
	g *Go
	linearGo *list.List //双向链表
	mutexLinearGo sync.Mutex
	mutexExecution sync.Mutex
}

//通道的大小为0的时候，阻塞式
//
func New(l int) *Go {
	g := new(Go)
	//缓存为l的函数类型通道
	g.ChanCb = make(chan func(),l)
	return g
}

func (g *Go)Go(f func(),cb func()){
	g.pendingGo++
	go func() {
		defer func() {
			g.ChanCb <- cb
			if r := recover();r!= nil {
				buf := make([]byte,4096)
				l := runtime.Stack(buf,false)
				log.Printf("[error]%v: %s",r,buf[:l])
			}
		}()

		f()

	}()
}

func (g *Go)Cb(cb func()){
	defer func (){
		g.pendingGo--
		if r := recover();r!= nil {
			if r := recover();r!= nil {
				buf := make([]byte,4096)
				l := runtime.Stack(buf,false)
				log.Printf("[error]%v: %s",r,buf[:l])
			}
		}
		if cb != nil {
			cb()
		}
	}()
}

func (g *Go)Close(){
	for g.pendingGo > 0{
		g.Cb(<-g.ChanCb)
	}
}

func (g *Go)Idle()bool{
	return g.pendingGo ==0
}

func (g *Go)NewLinearContext()*LinearContext{
	c := new(LinearContext)
	c.g = g
	c.linearGo = list.New()
	return c
}

func (c *LinearContext)Go(f func(),cb func()){
	c.g.pendingGo++

	c.mutexLinearGo.Lock()
	c.linearGo.PushBack(&LinearGo{f: f, cb: cb})
	c.mutexLinearGo.Unlock()

	go func() {
		c.mutexExecution.Lock()
		defer c.mutexExecution.Unlock()

		c.mutexLinearGo.Lock()
		e := c.linearGo.Remove(c.linearGo.Front()).(*LinearGo)
		c.mutexLinearGo.Unlock()

		defer func() {
			c.g.ChanCb <- e.cb
			if r := recover();r!= nil {
				if r := recover();r!= nil {
					buf := make([]byte,4096)
					l := runtime.Stack(buf,false)
					log.Printf("[error]%v: %s",r,buf[:l])
				}
			}
		}()

		e.f()
	}()
}