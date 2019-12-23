package go_pool

import (
	"log"
	"math/rand"
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
	p := NewWorkerPoolAndRun(1000, 1100)
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
			randNum := rand.Intn(10) + 1000
			p.AdjustSize(uint16(randNum))
		}
		log.Println("start wait.....")
		p.WaitAndClose()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	for {
		log.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
		time.Sleep(1 * time.Second)
	}
}
