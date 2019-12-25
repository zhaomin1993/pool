package go_pool

import (
	"errors"
	"sync"
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

func newWorker() *worker {
	return &worker{jobQueue: make(chan job), stop: make(chan struct{})}
}
func (w *worker) run(wq chan *worker, onPanic func(msg interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(r)
				}
				close(w.stop)
				close(w.jobQueue)
				worker := newWorker()
				worker.run(wq, onPanic)
				wq <- worker
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

func (w *worker) close() {
	w.stop <- struct{}{}
	close(w.stop)
	close(w.jobQueue)
}

// --------------------------- WorkerPool ---------------------
type workerPool struct {
	maxNum      uint16
	aliveNum    uint16
	workerNum   uint16
	mux         sync.RWMutex
	closeOnce   sync.Once
	workerQueue chan *worker
	onPanic     func(msg interface{})
}

//创建协程池
func NewWorkerPool(workerNum, maxNum uint16) *workerPool {
	if workerNum > maxNum {
		workerNum = maxNum
	}
	return &workerPool{
		maxNum:      maxNum,
		workerNum:   workerNum,
		mux:         sync.RWMutex{},
		closeOnce:   sync.Once{},
		workerQueue: make(chan *worker, maxNum),
	}
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
				err = errors.New("has no worker")
			}
		default:
			var worker *worker
			wp.mux.Lock()
			if wp.aliveNum < wp.workerNum {
				wp.aliveNum++
				wp.mux.Unlock()
				worker = newWorker()
				worker.run(wp.workerQueue, wp.onPanic)
			} else {
				wp.mux.Unlock()
				worker = <-wp.workerQueue
			}
			if worker != nil {
				worker.jobQueue <- job
			} else {
				err = errors.New("has no worker")
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
	defer wp.mux.RUnlock()
	return wp.aliveNum
}

//调整协程数
func (wp *workerPool) AdjustSize(workNum uint16) {
	if workNum > wp.maxNum {
		workNum = wp.maxNum
	}
	wp.mux.Lock()
	wp.workerNum = workNum
	if workNum < wp.aliveNum {
		for workNum < wp.aliveNum {
			wp.aliveNum--
			worker := <-wp.workerQueue
			worker.close()
		}
	}
	wp.mux.Unlock()
}

//关闭协程池
func (wp *workerPool) Close() {
	wp.closeOnce.Do(func() {
		wp.AdjustSize(0)
		close(wp.workerQueue)
	})
}
