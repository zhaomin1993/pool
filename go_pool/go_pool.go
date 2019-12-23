package go_pool

import (
	"sync"
)

// --------------------------- Job ---------------------
type Job interface {
	Do() //不允许发生panic也不允许永远阻塞,代码要求高
}

// --------------------------- Worker ---------------------
type worker struct {
	jobQueue chan Job
	stop     chan struct{}
}

func newWorker() worker {
	return worker{jobQueue: make(chan Job), stop: make(chan struct{})}
}
func (w *worker) run(wq chan *worker) {
	wq <- w
	go func() {
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
type WorkerPool struct {
	maxWoker    uint16
	workerNum   uint16
	mux         *sync.RWMutex
	stop        chan struct{}
	jobQueue    chan Job
	workerQueue chan *worker
}

//创建协程池并启动
func NewWorkerPoolAndRun(workerNum, maxWoker uint16) *WorkerPool {
	wp := NewWorkerPool(workerNum, maxWoker)
	wp.Run()
	return wp
}

//创建协程池
func NewWorkerPool(workerNum, maxWoker uint16) *WorkerPool {
	if workerNum > maxWoker {
		workerNum = maxWoker
	}
	return &WorkerPool{
		maxWoker:    maxWoker,
		workerNum:   workerNum,
		mux:         &sync.RWMutex{},
		stop:        make(chan struct{}),
		jobQueue:    make(chan Job),
		workerQueue: make(chan *worker, maxWoker),
	}
}

//启动协程池
func (wp *WorkerPool) Run() {
	//初始化worker
	wp.mux.RLock()
	workerlen := int(wp.workerNum)
	for i := 0; i < workerlen; i++ {
		worker := newWorker()
		worker.run(wp.workerQueue)
	}
	wp.mux.RUnlock()
	// 循环获取可用的worker,往worker中写job
	go func() {
		for {
			select {
			case job, _ := <-wp.jobQueue:
				worker := <-wp.workerQueue
				if job != nil {
					worker.jobQueue <- job
				}
			case <-wp.stop:
				return
			}
		}
	}()
}

//协程池接收任务
func (wp *WorkerPool) Accept(job Job) {
	wp.jobQueue <- job
}

//调整协程数
func (wp *WorkerPool) AdjustSize(workNum uint16) {
	if workNum > wp.maxWoker {
		workNum = wp.maxWoker
	}
	wp.mux.Lock()
	oldNum := wp.workerNum
	wp.workerNum = workNum
	if workNum > oldNum {
		for i := oldNum; i < workNum; i++ {
			worker := newWorker()
			worker.run(wp.workerQueue)
		}
	} else if workNum < oldNum {
		for i := workNum; i < oldNum; i++ {
			worker := <-wp.workerQueue
			worker.close()
		}
	}
	wp.mux.Unlock()
}

//等待协程池完成工作后关闭
func (wp *WorkerPool) WaitAndClose() {
	wp.AdjustSize(0)
	wp.Close()
}

//关闭协程池，但需要一点时间,如果是直接调用,发送Job的协程需要recover()
func (wp *WorkerPool) Close() {
	wp.stop <- struct{}{}
	close(wp.stop)
	if wp.workerNum != 0 {
		for worker := range wp.workerQueue {
			worker.close()
		}
	}
	close(wp.workerQueue)
	close(wp.jobQueue)
}
