package go_pool

import (
	"errors"
	"sync"

	"github.com/zhaomin1993/pool/internal"
)

// worker 工人
type worker struct {
	jobQueue chan internal.Job
	stop     chan struct{}
}

// newWorker 新建一个工人
func newWorker() *worker {
	return &worker{jobQueue: make(chan internal.Job), stop: make(chan struct{})}
}

// run 工人开始工作
func (w *worker) run(wq chan<- *worker, onPanic func(msg interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.run(wq, onPanic)
				wq <- w
				if onPanic != nil {
					onPanic(r)
				}
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

// close 关闭并回收工人
func (w *worker) close() {
	w.stop <- struct{}{}
	close(w.stop)
	close(w.jobQueue)
}

// workerPool 工厂
type workerPool struct {
	closed      bool
	maxNum      uint16
	aliveNum    uint16
	workerNum   uint16
	mux         sync.RWMutex
	closeOnce   sync.Once
	workerQueue chan *worker
	onPanic     func(msg interface{})
}

// NewWorkerPool 创建工厂
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

// OnPanic 设置工作中出问题时的处理方法
func (wp *workerPool) OnPanic(onPanic func(msg interface{})) {
	wp.onPanic = onPanic
}

// Accept 工厂接收工作任务
func (wp *workerPool) Accept(job internal.Job) (err error) {
	if job == nil {
		err = errors.New("job can not be nil")
		return
	}
	wp.mux.Lock()
	select {
	case worker := <-wp.workerQueue:
		wp.mux.Unlock()
		if worker != nil {
			worker.jobQueue <- job
		} else {
			err = errors.New("worker pool has been closed")
		}
	default:
		var worker *worker
		if wp.aliveNum == wp.workerNum {
			wp.mux.Unlock()
			worker = <-wp.workerQueue
		} else if wp.aliveNum < wp.workerNum {
			wp.aliveNum++
			wp.mux.Unlock()
			worker = newWorker()
			worker.run(wp.workerQueue, wp.onPanic)
		} else {
			wp.mux.Unlock()
			panic("worker number less than alive number")
		}
		if worker != nil {
			worker.jobQueue <- job
		} else {
			err = errors.New("worker pool has been closed")
		}
	}
	return
}

// Len 工厂活跃工人数
func (wp *workerPool) Len() uint16 {
	wp.mux.RLock()
	num := wp.aliveNum
	wp.mux.RUnlock()
	return num
}

// AdjustSize 调整工厂工人数量
func (wp *workerPool) AdjustSize(workNum uint16) {
	wp.mux.Lock()
	if wp.closed {
		wp.mux.Unlock()
		return
	}
	if workNum > wp.maxNum {
		workNum = wp.maxNum
	}
	oldNum := wp.workerNum
	wp.workerNum = workNum
	if oldNum == 0 && workNum > 0 {
		wp.aliveNum++
		worker := newWorker()
		worker.run(wp.workerQueue, wp.onPanic)
		wp.workerQueue <- worker
	}
	for workNum < wp.aliveNum {
		wp.aliveNum--
		worker := <-wp.workerQueue
		worker.close()
	}
	wp.mux.Unlock()
}

// Pause 工厂暂停工作
func (wp *workerPool) Pause() {
	wp.AdjustSize(0)
}

// Continue 工厂继续工作
func (wp *workerPool) Continue(num uint16) {
	wp.AdjustSize(num)
}

// Close 关闭并回收工厂
func (wp *workerPool) Close() {
	wp.closeOnce.Do(func() {
		wp.mux.Lock()
		wp.closed = true
		wp.workerNum = 0
		wp.maxNum = 0
		for 0 < wp.aliveNum {
			wp.aliveNum--
			worker := <-wp.workerQueue
			worker.close()
		}
		close(wp.workerQueue)
		wp.mux.Unlock()
	})
}
