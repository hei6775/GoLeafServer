# Golang Leaf框架阅读笔记

结合之前的游戏项目，研究Leaf的前世三生，主要是为了加强对项目的理解，以及源码的学习。

该仓库使用go的moudle作为管理，并将依赖注入vendor文件夹

# 基本介绍

看文档可以发现Leaf有很多设计亮点，像是：

* 能够恢复 goroutine 运行过程中的错误
* 在启动和结束时，保证模块的初始化同步性
* 重新设计的Go机制可以避免一些同步操作
* 和原生的定时器（`func AfterFunc(d Duration, f func()) *Timer`）相比，可以在当前goroutine下执行函数f
* 模块功能区分，并通过一套基于channel的RPC进行通信

# 浅析

### [Leaf的Go机制浅析（完成）](https://github.com/hei6775/GoLeafServer/blob/master/LeafNotes/LeafGo.MD)

### [Leaf的定时器浅析（待完成）]()

### [Leaf的Log模块浅析 （待完成）]()

### [Leaf的ChanRPC模块浅析（完成）](https://github.com/hei6775/GoLeafServer/blob/master/LeafNotes/ChanRPC.MD)

### [Leaf的Gate模块浅析 （待完成）]()

### [Leaf的NetWork模块浅析（完成）](https://github.com/hei6775/GoLeafServer/blob/master/LeafNotes/NetWork.MD)

### [Leaf的Moudle模块浅析（待完成）]()

### [Leaf的游戏配置文件读取浅析（待完成）]()