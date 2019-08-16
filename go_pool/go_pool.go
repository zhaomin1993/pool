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
func (w *worker) run(wq chan *worker, wg *sync.WaitGroup) {
	wq <- w
	wg.Add(1)
	go func() {
		for {
			select {
			case job := <-w.jobQueue:
				if job != nil {
					job.Do()
					wq <- w
				} else {
					wg.Done()
					close(w.stop)
					close(w.jobQueue)
					return
				}
			case <-w.stop:
				return
				//runtime.Goexit()
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
	workerlen   uint16
	stopSignal  uint16
	wg          *sync.WaitGroup
	stop        chan struct{}
	jobQueue    chan Job
	workerQueue chan *worker
}

//创建协程池并启动
func NewWorkerPoolAndRun(workerlen uint16) *WorkerPool {
	wp := NewWorkerPool(workerlen)
	wp.Run()
	return wp
}

//创建协程池
func NewWorkerPool(workerlen uint16) *WorkerPool {
	return &WorkerPool{
		workerlen:   workerlen,
		stopSignal:  0,
		wg:          &sync.WaitGroup{},
		stop:        make(chan struct{}),
		jobQueue:    make(chan Job),
		workerQueue: make(chan *worker, workerlen),
	}
}

//启动协程池
func (wp *WorkerPool) Run() {
	//初始化worker
	workerlen := int(wp.workerlen)
	for i := 0; i < workerlen; i++ {
		worker := newWorker()
		worker.run(wp.workerQueue, wp.wg)
	}
	// 循环获取可用的worker,往worker中写job
	go func() {
		for {
			select {
			case job, ok := <-wp.jobQueue:
				worker := <-wp.workerQueue
				worker.jobQueue <- job
				if job == nil && !ok {
					wp.stopSignal++
					if wp.stopSignal == wp.workerlen {
						return
					}
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

//等待协程池完成工作后关闭
func (wp *WorkerPool) WaitAndClose() {
	wp.wait()
	wp.Close()
}

//关闭协程池，但需要一点时间,如果不是在wait()后调用,发送Job的协程需要recover()
func (wp *WorkerPool) Close() {
	if wp.stopSignal == 0 {
		wp.stop <- struct{}{}
		workerlen := int(wp.workerlen)
		for i := 0; i < workerlen; i++ {
			select {
			case worker := <-wp.workerQueue:
				worker.close()
			}
		}
		select {
		case _, ok := <-wp.jobQueue:
			if ok {
				close(wp.jobQueue)
			}
		default:
			close(wp.jobQueue)
		}
	}
	close(wp.stop)
	close(wp.workerQueue)
}

//等待协程池工作完成
func (wp *WorkerPool) wait() {
	close(wp.jobQueue)
	wp.wg.Wait()
}
