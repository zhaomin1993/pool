package go_pool

import (
	"github.com/panjf2000/ants"
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 1000000
	BenchParam         = 10
	BenchGoSize        = 20000
	DefaultExpiredTime = 10 * time.Second
)

type obj struct{}

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
	sema := make(chan struct{}, BenchGoSize)

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

//go test -bench=BenchmarkAntsPool -benchmem=true -run=none
//BenchmarkAntsPool-8                            2         593923650 ns/op        21303400 B/op    1066102 allocs/op
func BenchmarkAntsPool(b *testing.B) {
	var wg sync.WaitGroup
	p, _ := ants.NewPool(BenchGoSize, ants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
}

//go test -bench=BenchmarkGoPool -benchmem=true -run=none
//BenchmarkGoPool-8                      2         571007850 ns/op         3657388 B/op      46307 allocs/op
func BenchmarkGoPool(b *testing.B) {
	p := NewWorkerPool(uint16(BenchGoSize), uint16(BenchGoSize), DefaultExpiredTime)

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
	sema := make(chan struct{}, BenchGoSize)
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

//BenchmarkGoPoolThroughput-8            2         581976150 ns/op         3570988 B/op      46128 allocs/op
func BenchmarkGoPoolThroughput(b *testing.B) {
	p := NewWorkerPool(uint16(BenchGoSize), uint16(BenchGoSize), DefaultExpiredTime)
	defer p.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Accept(obj{})
		}
	}
	b.StopTimer()
}

//BenchmarkAntsPoolThroughput-8                  2         550545450 ns/op         2504716 B/op      41251 allocs/op
func BenchmarkAntsPoolThroughput(b *testing.B) {
	p, _ := ants.NewPool(BenchGoSize, ants.WithExpiryDuration(DefaultExpiredTime))
	defer p.Release()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = p.Submit(demoFunc)
		}
	}
	b.StopTimer()
}
