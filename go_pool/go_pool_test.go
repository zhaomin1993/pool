package go_pool

import (
	"fmt"
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
	if s.Num%2 == 0 {
		panic(s.Num)
	}
	time.Sleep(time.Millisecond * 100)
}

//go test -v -test.run TestWorkerPool_Run
func TestWorkerPool_Run(t *testing.T) {
	log.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
	p := NewWorkerPool(1000, 1100, time.Second*5)
	p.OnPanic(func(msg interface{}) {
		//log.Println("error:", msg)
	})
	datanum := 100 * 100 * 100
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		start := time.Now()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
			if i%10000 == 0 {
				log.Println("send num:", i)
			}
			randNum := rand.Intn(10) + 1000
			p.AdjustSize(uint16(randNum))
		}
		log.Println("start wait.....")
		//p.Close()
		log.Printf("cost time:%s\n", time.Since(start).String())
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	go func() {
		time.Sleep(time.Second * 62)
		log.Println("start again.....------------------------------")
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
			if i%10000 == 0 {
				log.Println("send num:", i)
			}
			randNum := rand.Intn(10) + 1000
			p.AdjustSize(uint16(randNum))
		}
		p.Close()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
			if i%10000 == 0 {
				log.Println("send num:", i)
			}
			randNum := rand.Intn(10) + 1000
			p.AdjustSize(uint16(randNum))
		}
	}()
	for {
		time.Sleep(1 * time.Second)
		log.Printf("runtime.NumGoroutine() :%d\n", runtime.NumGoroutine())
	}
}

//go test -v -test.run TestWorkerPool_Close
func TestWorkerPool_Close(t *testing.T) {
	log.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
	p := NewWorkerPool(1000, 1100, time.Second)
	p.OnPanic(func(msg interface{}) {
		//log.Println("error:", msg)
	})
	datanum := 100 * 100 * 100
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
		}
		log.Println("start wait.....")
		p.Close()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	go func() {
		rand.Seed(time.Now().Unix())
		randNum := rand.Intn(3) + 1
		time.Sleep(time.Second * time.Duration(randNum))
		p.Close()
	}()
	go func() {
		for {
			time.Sleep(time.Second)
			randNum := rand.Intn(10) + 1000
			p.AdjustSize(uint16(randNum))
		}
	}()
	for {
		time.Sleep(1 * time.Second)
		log.Printf("runtime.NumGoroutine() :%d\n", runtime.NumGoroutine())
	}
}

//go test -v -test.run TestWorkerPool_AdjustSize
func TestWorkerPool_AdjustSize(t *testing.T) {
	log.Println("runtime.NumGoroutine() :", runtime.NumGoroutine())
	p := NewWorkerPool(1000, 1100, time.Second)
	p.OnPanic(func(msg interface{}) {
		//log.Println("error:", msg)
	})
	datanum := 100 * 100 * 100
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
		}
		log.Println("start wait.....")
		p.Close()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
		}
		log.Println("start wait.....")
		p.Close()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println(r)
			}
		}()
		for i := 1; i <= datanum; i++ {
			sc := &Score{Num: i}
			if err := p.Accept(sc); err != nil {
				fmt.Println("err:\t", err)
				break
			}
		}
		log.Println("start wait.....")
		p.Close()
		log.Println("stop over.....")
		log.Println("the last runtime.NumGoroutine() :", runtime.NumGoroutine())
	}()
	go func() {
		num := 0
		for {
			time.Sleep(time.Second)
			num++
			if num%2 == 0 {
				p.AdjustSize(0)
			} else {
				p.AdjustSize(1000)
			}
		}
	}()
	for {
		time.Sleep(1 * time.Second)
		log.Printf("runtime.NumGoroutine() :%d\n", runtime.NumGoroutine())
	}
}
