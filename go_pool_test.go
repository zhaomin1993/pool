package pool

import (
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
		log.Println("num:", s.Num)
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
				log.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			p.Accept(sc)
		}
		p.Wait()
		p.Close()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	for {
		log.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
		time.Sleep(1 * time.Second)
	}
	//time.Sleep(5 * time.Second)

}
