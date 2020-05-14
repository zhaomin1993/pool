package go_pool

import (
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 1000000
	BenchParam         = 10
	BenchAntsSize      = 20000
)

type obj struct {}

func (obj) Do() {
	demoFunc()
}

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}

//go test -bench=BenchmarkGoroutines -benchmem=true -run=none
//BenchmarkGoroutines-8                          1        3276239400 ns/op        537844448 B/op   1997684 allocs/op
func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

//go test -bench=BenchmarkSemaphore -benchmem=true -run=none
//BenchmarkSemaphore-8                           2         651259850 ns/op        64292344 B/op    1001274 allocs/op
func BenchmarkSemaphore(b *testing.B) {
	var wg sync.WaitGroup
	sema := make(chan struct{}, BenchAntsSize)

	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				demoFunc()
				<-sema
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

//go test -bench=BenchmarkGoPool -benchmem=true -run=none
//BenchmarkGoPool-8                            2         638791000 ns/op         3654000 B/op      48113 allocs/op
func BenchmarkGoPool(b *testing.B) {
	p := NewWorkerPool(uint16(BenchAntsSize), uint16(BenchAntsSize))

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Accept(obj{})
		}
	}
	p.Close()
	b.StopTimer()
}

//BenchmarkGoroutinesThroughput-8                4         264044400 ns/op        63998256 B/op     999972 allocs/op
func BenchmarkGoroutinesThroughput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			go demoFunc()
		}
	}
}

//BenchmarkSemaphoreThroughput-8                 2         631312300 ns/op        64034400 B/op    1000145 allocs/op
func BenchmarkSemaphoreThroughput(b *testing.B) {
	sema := make(chan struct{}, BenchAntsSize)
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				demoFunc()
				<-sema
			}()
		}
	}
}

//BenchmarkGoPoolThroughput-8                  2         644277400 ns/op         8187200 B/op      61301 allocs/op
func BenchmarkGoPoolThroughput(b *testing.B) {
	p := NewWorkerPool(uint16(BenchAntsSize), uint16(BenchAntsSize))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Accept(obj{})
		}
	}
	b.StopTimer()
}