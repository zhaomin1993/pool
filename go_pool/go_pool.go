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
func (w *worker) run(wq chan<- *worker, onPanic func(msg interface{})) {
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

func (w *worker) stopRun() {
	w.stop <- struct{}{}
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
	blockAccept uint16
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
		switch {
		case wp.aliveNum == wp.workerNum:
			if wp.workerNum == wp.workerSize {
				if wp.workerSize == 0 {
					wp.blockAccept++
				}
				wp.mux.Unlock()
				worker := <-wp.workerQueue
				if worker != nil {
					worker.jobQueue <- job
				} else {
					err = errors.New("worker pool has been closed")
				}
				return
			}
			wp.workerNum = wp.workerSize
			wp.mux.Unlock()
			err = wp.Accept(job)
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
	return
}

//获取协程数
func (wp *workerPool) Len() uint16 {
	wp.mux.RLock()
	num := wp.aliveNum
	wp.mux.RUnlock()
	return num
}

//调整协程池大小
func (wp *workerPool) AdjustSize(workSize uint16) {
	wp.mux.Lock()
	if wp.closed {
		wp.mux.Unlock()
		return
	}
	if workSize > wp.maxSize {
		workSize = wp.maxSize
	}
	if wp.workerNum > workSize {
		wp.workerNum = workSize
	}
	oldSize := wp.workerSize
	wp.workerSize = workSize
	if oldSize == 0 && workSize > 0 && wp.blockAccept > 0 {
		times := wp.blockAccept
		if workSize < wp.blockAccept {
			times = workSize
		}
		wp.blockAccept = 0
		for i := uint16(0); i < times; i++ {
			wp.aliveNum++
			wp.workerNum++
			worker := wp.workers.Get().(*worker)
			worker.run(wp.workerQueue, wp.onPanic)
			wp.workerQueue <- worker
		}
	}
	for workSize < wp.aliveNum {
		wp.aliveNum--
		worker := <-wp.workerQueue
		worker.stopRun()
		wp.workers.Put(worker)
	}
	wp.mux.Unlock()
}

//暂停
func (wp *workerPool) Pause() {
	wp.AdjustSize(0)
}

//继续
func (wp *workerPool) Continue(num uint16) {
	wp.AdjustSize(num)
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
		for {
			select {
			case <-ticker.C:
				wp.mux.Lock()
				length := len(wp.workerQueue)
				if 1 < length && uint16(length) <= wp.aliveNum {
					num := wp.aliveNum - uint16(length>>1)
					wp.workerNum = num
					for num < wp.aliveNum {
						wp.aliveNum--
						worker := <-wp.workerQueue
						worker.stopRun()
						wp.workers.Put(worker)
					}
				}
				wp.mux.Unlock()
			case <-wp.stopAuto:
				ticker.Stop()
				return
			}
		}
	}()
}
