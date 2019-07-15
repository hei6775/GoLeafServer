package chanrpc

import (
	"fmt"
	"sync"
	"testing"
)

func f0(args []interface{}) {
	fmt.Println("bbbb f0")
}

func f1(args []interface{}) interface{} {
	fmt.Println("bbbbb f1")
	return "f1"
}

func f2(args []interface{}) []interface{} {
	fmt.Println("bbbbb f2")
	return []interface{}{"f2", "i am f2"}
}

func TestChanrpc(t *testing.T) {

	s := NewServer(10)

	var wg sync.WaitGroup
	wg.Add(1)

	//goroutine

	go func() {
		s.Register("f0", f0)
		s.Register("f1", f1)
		s.Register("f2", f2)
		fmt.Println("ok register")
		wg.Done()

		for {
			s.Exec(<-s.ChanCall)
		}
	}()

	wg.Wait()
	wg.Add(1)

	go func() {
		c := s.Open(10)

		//sync 同步
		err := c.Call0("f0")
		if err != nil {
			fmt.Println(err)
		}

		//r1, err := c.Call1("f1")
		//if err != nil {
		//	fmt.Println(err)
		//} else {
		//	fmt.Println(r1)
		//}
		//
		//rn, err := c.CallN("f2")
		//if err != nil {
		//	fmt.Println(err)
		//} else {
		//	fmt.Println(rn...)
		//}

		//asyn 异步
		c.AsynCall("f0", func(err error) {
			if err != nil {
				fmt.Println(err)
			}
		})

		c.AsynCall("f1", func(ret interface{}, err error) {
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(ret)
			}
		})

		c.AsynCall("f2", func(ret []interface{}, err error) {
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(ret...)
			}
		})
		c.Cb(<-c.chanAsynRet)
		c.Cb(<-c.chanAsynRet)
		c.Cb(<-c.chanAsynRet)

		s.Go("f1")

		wg.Done()
	}()

	wg.Wait()
}
