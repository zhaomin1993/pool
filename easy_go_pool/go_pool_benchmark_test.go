package go_pool

import (
	"github.com/panjf2000/ants"
	"sync"
	"sync/atomic"
	"testing"
)

const (
	RunTimes    = 1000000
	BenchParam  = 10
	BenchGoSize = 20000
)

type obj int32

func (this obj) Do() {
	myFunc(int32(this))
}

var sum int32

func myFunc(i interface{}) {
	n := i.(int32)
	atomic.AddInt32(&sum, n)
	// fmt.Printf("run with %d\n", n)
}

//go test -bench=BenchmarkGoroutines -benchmem=true -run=none
//BenchmarkGoroutines-8                          1        3276239400 ns/op        537844448 B/op   1997684 allocs/op
func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				myFunc(int32(j))
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
	sema := make(chan struct{}, BenchGoSize)

	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				myFunc(int32(j))
				<-sema
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

//go test -bench=BenchmarkAntsPool -benchmem=true -run=none
//BenchmarkAntsPool-8                            2         587444950 ns/op        20669048 B/op    1059622 allocs/op
func BenchmarkAntsPool(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(BenchGoSize, func(i interface{}) {
		myFunc(i)
		wg.Done()
	})
	defer p.Release()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			p.Invoke(int32(j))
		}
		wg.Wait()
	}
	b.StopTimer()
}

//go test -bench=BenchmarkGoPool -benchmem=true -run=none
//BenchmarkGoPool-8                      2         564994050 ns/op         3521408 B/op      45269 allocs/op
func BenchmarkGoPool(b *testing.B) {
	p := NewWorkerPool(uint16(BenchGoSize), uint16(BenchGoSize))
	defer p.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		p.AdjustSize(uint16(BenchGoSize))
		for j := 0; j < RunTimes; j++ {
			_ = p.Accept(obj(j))
		}
		p.AdjustSize(0)
	}
	b.StopTimer()
}

//BenchmarkGoroutinesThroughput-8                4         264044400 ns/op        63998256 B/op     999972 allocs/op
func BenchmarkGoroutinesThroughput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			go myFunc(int32(j))
		}
	}
}

//BenchmarkSemaphoreThroughput-8                 2         631312300 ns/op        64034400 B/op    1000145 allocs/op
func BenchmarkSemaphoreThroughput(b *testing.B) {
	sema := make(chan struct{}, BenchGoSize)
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			sema <- struct{}{}
			go func() {
				myFunc(int32(j))
				<-sema
			}()
		}
	}
}

//BenchmarkGoPoolThroughput-8            2         562498700 ns/op         3385600 B/op      43775 allocs/op
func BenchmarkGoPoolThroughput(b *testing.B) {
	p := NewWorkerPool(uint16(BenchGoSize), uint16(BenchGoSize))
	defer p.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Accept(obj(j))
		}
	}
	b.StopTimer()
}

//BenchmarkAntsPoolThroughput-8                  2         552554100 ns/op         2588784 B/op      41928 allocs/op
func BenchmarkAntsPoolThroughput(b *testing.B) {
	p, _ := ants.NewPoolWithFunc(BenchGoSize, func(i interface{}) {
		myFunc(i)
	})
	defer p.Release()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Invoke(int32(j))
		}
	}
	b.StopTimer()
}
