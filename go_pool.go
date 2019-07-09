package pool

import (
	"sync"
)

/*高并发任务*/
/*
1、工人同一时间只能处理一个工作，每个工人处理工作与其他工人无关，所以单独开启协程
2、工厂开工（WorkerPool.Run()）则工人加入工人队列开始工作（Worker.Run()）
3、创建任务并循环加入WorkerPool.JobQueue（工厂工作队列），让工人队列（WorkerPool.WorkerQueue）中的工人（协程）处理
4、从WorkerPool.JobQueue中接受任务，并以此分发给工人队列（WorkerPool.WorkerQueue）中的工人,工人工作完成后再返回队列
5、工人接收到任务执行Do（）
6、关闭协程池先停止工人从工作队列中拿取Job，再遍历协程池中的工人队列，让工人停止工作，由于stop是非缓存通道，所以close（）需要花费一定时间
7、关闭协程池的工作队列，然后工人根据拿到的Job是否为nil来判断所有任务是否已经完成，完成则退出
8、wait()等待协程池完工，既关闭工作队列，让工人去判断是否还有工作，没有则下班
*/
//核心思想：固定数量的goroutine for循环处理一同个channel中的数据
// --------------------------- Job ---------------------
type Job interface {
	Do()
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
			case <-w.stop: //由于select的伪随机性，stop不一定会执行，导致stop没有即时性
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
	JobQueue    chan Job
	workerQueue chan *worker
}

func NewWorkerPool(workerlen uint16) *WorkerPool {
	return &WorkerPool{
		workerlen:   workerlen,
		stopSignal:  0,
		wg:          &sync.WaitGroup{},
		stop:        make(chan struct{}),
		JobQueue:    make(chan Job),
		workerQueue: make(chan *worker, workerlen),
	}
}
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
			case job, ok := <-wp.JobQueue:
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
		close(wp.JobQueue)
	}
	close(wp.stop)
	close(wp.workerQueue)
}

//等待协程池工作完成
func (wp *WorkerPool) Wait() {
	close(wp.JobQueue)
	wp.wg.Wait()
}
