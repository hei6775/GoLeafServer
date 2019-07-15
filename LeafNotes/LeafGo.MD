# Leaf的Go机制

> 官方文档：Leaf Go
>
>    善用 goroutine 能够充分利用多核资源，Leaf 提供的 Go 机制解决了
>      原生 goroutine 存在的一些问题：
>      
>    能够恢复 goroutine 运行过程中的错误
>
>    游戏服务器会等待所有 goroutine 执行结束后才关闭
>
>    非常方便的获取 goroutine 执行的结果数据
>
>    在一些特殊场合保证 goroutine 按创建顺序执行
>
>    第一个函数会被放置在一个新创建的 goroutine 中执行，在其执行完成之后，第二个函数会在当前 goroutine 中被执行。由此，我们可以看到变量 res 同一时刻总是只被一个 goroutine 访问，这就避免了同步机制的使用。Go 的设计使得 CPU 得到充分利用，避免操作阻塞当前 goroutine，同时又无需为共享资源同步而忧心。


对于Leaf的Go机制，代码并不长，所以理解起来并不难，但是它在Leaf中确实保证了一定的同步，避免了一些额外的繁琐的操作。
项目中一般用它来保证两个函数对相同变量的顺序访问，避免出现竞争。

Go机制以Leaf自定义的Go结构体为核心：
```go
//Go结构体
type Go struct {
	ChanCb chan func() //回调函数通道
	pendingGo int //等待的go的数量
}

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
```
对于`New(l)`中如何设置合适的`channel`容量，建议参考[Go Behavior Of Channels](https://github.com/hei6775/HeiBlogs/blob/master/GoVersion/GoBehaviorOfChannels.md)
以及项目需求。主要看消息的传递保障性。

可以发现Go机制控制了并发的Go数量，每次在当前协程新开一个`goroutine`新开一个Go的任务，就会在`pending`中++，
每次完成一个Go的任务就会对应的减少`pending`。当然代码也比较简洁，Go机制第一个函数参数会在新开的`goroutine`中
完成，并不会阻塞当前`goroutine`，并且在完成后通过调用`Close()`方法或者直接调用`Cb()`函数，会在当前`goroutine`
完成第二个函数参数。

在这里，理论上等到通道消息传过来是会发生阻塞的，但是在Leaf框架中，集成到`skeleton`中通过`select`同时监听多个消息通道，
从而避免阻塞第一个任务阻塞当前协程。
```go
func (s *Skeleton) Run(closeSig chan bool) {
	for {
		select {
		case <-closeSig:
			s.commandServer.Close()
			s.server.Close()
			for !s.g.Idle() || !s.client.Idle() {
				s.g.Close()
				s.client.Close()
			}
			return
		case ri := <-s.client.ChanAsynRet:
			s.client.Cb(ri)
		case ci := <-s.server.ChanCall:
			s.server.Exec(ci)
		case ci := <-s.commandServer.ChanCall:
			s.commandServer.Exec(ci)
		case cb := <-s.g.ChanCb:
		    //就是这里
			s.g.Cb(cb)
		case t := <-s.dispatcher.ChanTimer:
			t.Cb()
		}
	}
}
```



