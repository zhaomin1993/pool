package pool

import (
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"
)

type Score struct {
	Num int
}

func (s *Score) Do() {
	if s.Num%10000 == 0 {
		fmt.Println("num:", s.Num)
	}
	time.Sleep(time.Millisecond * 10)
}

//go test -v -test.run TestWorkerPool_Run
func TestWorkerPool_Run(t *testing.T) {
	p := NewWorkerPool(1000)
	p.Run()
	datanum := 100 * 100 * 100
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			p.JobQueue <- sc
		}
		p.Wait()
		p.Stop()
		log.Println("stop over.....")
		fmt.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	for {
		fmt.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
		time.Sleep(1 * time.Second)
	}
	//time.Sleep(5 * time.Second)

}
