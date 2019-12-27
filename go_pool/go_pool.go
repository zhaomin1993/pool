package go_pool

import (
	"errors"
	"sync"
	"time"
)

// --------------------------- Job ---------------------
type job interface {
	Do() //不允许永远阻塞,代码要求高
}

// --------------------------- Worker ---------------------
type worker struct {
	jobQueue chan job
	stop     chan struct{}
}

//创建一个工人
func newWorker() *worker {
	return &worker{jobQueue: make(chan job), stop: make(chan struct{})}
}

//工人进入工作状态
func (w *worker) run(wq chan *worker, onPanic func(msg interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(r)
				}
				w.run(wq, onPanic)
				wq <- w
			}
		}()
		for {
			select {
			case job := <-w.jobQueue:
				job.Do()
				wq <- w
			case <-w.stop:
				return
			}
		}
	}()
}

//回收工人
func (w *worker) close() {
	w.stop <- struct{}{}
	close(w.stop)
	close(w.jobQueue)
}

// --------------------------- WorkerPool ---------------------
type workerPool struct {
	closed      bool
	maxSize     uint16
	aliveNum    uint16
	workerNum   uint16
	workerSize  uint16
	workers     sync.Pool
	mux         sync.RWMutex
	closeOnce   sync.Once
	workerQueue chan *worker
	stopAuto    chan struct{}
	onPanic     func(msg interface{})
}

//创建协程池
func NewWorkerPool(workerNum, maxSize uint16, interval time.Duration) *workerPool {
	wp := &workerPool{
		maxSize:     maxSize,
		workerNum:   workerNum,
		workerSize:  workerNum,
		mux:         sync.RWMutex{},
		closeOnce:   sync.Once{},
		workerQueue: make(chan *worker, maxSize),
		stopAuto:    make(chan struct{}),
		workers:     sync.Pool{},
	}
	wp.workers.New = func() interface{} {
		return newWorker()
	}
	wp.autoCutCap(interval)
	return wp
}

func (wp *workerPool) OnPanic(onPanic func(msg interface{})) {
	wp.onPanic = onPanic
}

//协程池接收任务
func (wp *workerPool) Accept(job job) (err error) {
	if job != nil {
		select {
		case worker := <-wp.workerQueue:
			if worker != nil {
				worker.jobQueue <- job
			} else {
				err = errors.New("worker pool has been closed")
			}
		default:
			wp.mux.Lock()
			switch {
			case wp.closed:
				wp.mux.Unlock()
				err = errors.New("worker pool has been closed")
			case wp.aliveNum == wp.workerNum:
				if wp.workerSize == 0 {
					wp.mux.Unlock()
					err = errors.New("has no worker")
					return
				}
				if wp.workerNum == wp.workerSize {
					wp.mux.Unlock()
					worker := <-wp.workerQueue
					if worker != nil {
						worker.jobQueue <- job
					} else {
						err = errors.New("worker pool has been closed")
					}
					return
				} else {
					wp.mux.Unlock()
					wp.adjustNum(wp.workerSize)
					err = wp.Accept(job)
					return
				}
			case wp.aliveNum < wp.workerNum:
				wp.aliveNum++
				wp.mux.Unlock()
				worker := wp.workers.Get().(*worker)
				worker.run(wp.workerQueue, wp.onPanic)
				worker.jobQueue <- job
			default:
				wp.mux.Unlock()
				panic("worker number less than alive number")
			}
		}
	} else {
		err = errors.New("job can not be nil")
	}
	return
}

//获取协程数
func (wp *workerPool) Cap() uint16 {
	wp.mux.RLock()
	num := wp.aliveNum
	wp.mux.RUnlock()
	return num
}

//调整协程池大小
func (wp *workerPool) AdjustSize(workSize uint16) {
	wp.mux.Lock()
	if workSize > wp.maxSize {
		workSize = wp.maxSize
	}
	if wp.workerNum > workSize {
		wp.workerNum = workSize
	}
	wp.workerSize = workSize
	for workSize < wp.aliveNum {
		wp.aliveNum--
		worker := <-wp.workerQueue
		worker.close()
		wp.workers.Put(worker)
	}
	wp.mux.Unlock()
}

//调整协程数
func (wp *workerPool) adjustNum(workNum uint16) {
	wp.mux.Lock()
	if workNum > wp.workerSize {
		workNum = wp.workerSize
	}
	wp.workerNum = workNum
	for workNum < wp.aliveNum {
		wp.aliveNum--
		worker := <-wp.workerQueue
		worker.close()
		wp.workers.Put(worker)
	}
	wp.mux.Unlock()
}

//关闭协程池
func (wp *workerPool) Close() {
	wp.closeOnce.Do(func() {
		wp.mux.Lock()
		wp.closed = true
		close(wp.stopAuto)
		wp.workerNum = 0
		wp.workerSize = 0
		wp.maxSize = 0
		for 0 < wp.aliveNum {
			wp.aliveNum--
			worker := <-wp.workerQueue
			worker.close()
		}
		close(wp.workerQueue)
		wp.mux.Unlock()
	})
}

//自动缩容
func (wp *workerPool) autoCutCap(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				length := len(wp.workerQueue)
				wp.mux.RLock()
				if wp.aliveNum == uint16(length) && length != 0 {
					wp.mux.RUnlock()
					wp.adjustNum(uint16(length / 2))
				} else {
					wp.mux.RUnlock()
				}
			case <-wp.stopAuto:
				return
			}
		}
	}()
}
