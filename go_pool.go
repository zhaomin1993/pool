package pool

import (
	"sync"
)

/*高并发任务*/
/*
1、工人同一时间只能处理一个工作，每个工人处理工作与其他工人无关，所以单独开启协程
2、工厂开工（WorkerPool.Run()）则工人加入工人队列开始工作（Worker.Run()）
3、创建任务并循环加入WorkerPool.JobQueue（工厂工作队列，这里的工作列队没有缓冲），让工人队列（WorkerPool.WorkerQueue）中的工人（协程）处理
4、从WorkerPool.JobQueue中接受任务，并以此分发给工人队列（WorkerPool.WorkerQueue）中的工人
5、工人接收到任务执行Do（）
6、关闭协程池先停止工人从工作队列中拿取Job，再遍历协程池中的工人队列，让工人停止工作，由于stop是非缓存通道，所以stop（）需要花费一定时间
7、关闭协程池的工作队列，然后工人根据拿到的Job是否为nil来判断所有任务是否已经完成，完成则退出
*/
//核心思想：固定数量的goroutine for循环处理一同个channel中的数据
// --------------------------- Job ---------------------
type Job interface {
	Do()
}

// --------------------------- Worker ---------------------
type Worker struct {
	JobQueue chan Job
	stop     chan struct{}
}

func NewWorker() Worker {
	return Worker{JobQueue: make(chan Job), stop: make(chan struct{})}
}
func (w *Worker) Run(wq chan *Worker, wg *sync.WaitGroup) {
	wq <- w
	wg.Add(1)
	go func() {
		for {
			select {
			case job := <-w.JobQueue:
				if job != nil {
					job.Do()
					wq <- w
				} else {
					wg.Done()
					close(w.stop)
					close(w.JobQueue)
					return
				}
			case <-w.stop: //由于select的伪随机性，stop不一定会执行，导致stop没有即时性
				return
				//runtime.Goexit()
			}
		}
	}()
}

func (w *Worker) Stop() {
	w.stop <- struct{}{}
	close(w.stop)
	close(w.JobQueue)
}

// --------------------------- WorkerPool ---------------------
type WorkerPool struct {
	workerlen   int
	stopSignal  int
	wg          *sync.WaitGroup
	stop        chan struct{}
	JobQueue    chan Job
	WorkerQueue chan *Worker
}

func NewWorkerPool(workerlen int) *WorkerPool {
	return &WorkerPool{
		workerlen:   workerlen,
		stopSignal:  0,
		wg:          &sync.WaitGroup{},
		stop:        make(chan struct{}),
		JobQueue:    make(chan Job, workerlen),
		WorkerQueue: make(chan *Worker, workerlen),
	}
}
func (wp *WorkerPool) Run() {
	//初始化worker
	for i := 0; i < wp.workerlen; i++ {
		worker := NewWorker()
		worker.Run(wp.WorkerQueue, wp.wg)
	}
	// 循环获取可用的worker,往worker中写job
	go func() {
		for {
			select {
			case job, ok := <-wp.JobQueue:
				worker := <-wp.WorkerQueue
				worker.JobQueue <- job
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

//关闭协程池，但需要一点时间,如果不是在wait()后调用,发送Job的协程需要recover()
func (wp *WorkerPool) Stop() {
	if wp.stopSignal == 0 {
		wp.stop <- struct{}{}
		for i := 0; i < wp.workerlen; i++ {
			select {
			case worker := <-wp.WorkerQueue:
				worker.Stop()
			}
		}
		close(wp.JobQueue)
	}
	close(wp.stop)
	close(wp.WorkerQueue)
}

//等待协程池工作完成
func (wp *WorkerPool) Wait() {
	close(wp.JobQueue)
	wp.wg.Wait()
}
