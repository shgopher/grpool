package grpool

import "sync"

// Gorouting instance which can accept client jobs
type worker struct {
	// worker 池：chan chan Job
	workerPool chan *worker
	// Job
	jobChannel chan Job
	// channel 传递信号通知
	stop chan struct{}
}

// worker 开始运作
func (w *worker) start() {
	// 新开启一个 goroutine 去运行 Job
	go func() {
		var job Job
		for {
			// worker free, add it to pool
			// 往 worker 池中发送 worker
			// 当select已经接收任务并且执行完毕的时候
			// 才会往 workerPool 发送 worker
			// 因为这代表它空闲了
			w.workerPool <- w

			select {
			// 从 job channel 中 接收 job 并执行
			case job = <-w.jobChannel:
				job()
				// 接收到 stop 信号之后，随机退出
			case <-w.stop:
				// 这里使用了信号量的相关操作，操作完成之后，v 操作
				// 这里的v操作就是释放资源：往channel中发送数据，等于增加数据
				// 这里的 p 操作就是获取资源：从channel中获取一个数据，等于 减少数据

				// 这里的操作就重新装填channel，保证如新；
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

// 创建一个 worker
// 这里面的 workerPool 实际上是同一个；
// 所以在创建的时候会将那唯一的一个 pool再次传入进行使用
func newWorker(pool chan *worker) *worker {
	return &worker{
		workerPool: pool,
		jobChannel: make(chan Job),
		stop:       make(chan struct{}),
	}
}

// 任务池
// Accepts jobs from clients, and waits for first free worker to deliver job
type dispatcher struct {
	// worker 池
	workerPool chan *worker
	// JobQueue
	jobQueue chan Job
	stop     chan struct{}
}

// 派遣任务
func (d *dispatcher) dispatch() {
	for {
		select {
		// 从 jobq中获取任务
		case job := <-d.jobQueue:
			// 从workerPool中获取一个空闲的worker
			worker := <-d.workerPool
			// 将 job 传入 worker 的 jobChannel
			worker.jobChannel <- job
		case <-d.stop:
			// 这个case 是指的当stop的时候，全部的worker都退出
			for i := 0; i < cap(d.workerPool); i++ {
				worker := <-d.workerPool

				// 这个操作是提供信号，并且恢复原样
				worker.stop <- struct{}{}
				<-worker.stop
			}

			d.stop <- struct{}{}
			return
		}
	}
}

// workerpool 是固定的那一个，
func newDispatcher(workerPool chan *worker, jobQueue chan Job) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		stop:       make(chan struct{}),
	}
	// 创建的时候，直接运行 worker pool
	for i := 0; i < cap(d.workerPool); i++ {
		worker := newWorker(d.workerPool)
		worker.start()
	}
	// 这里是值得 从jobqueue中获取任务 然后唤醒woker pool 中的woker
	go d.dispatch()
	return d
}

// Represents user request, function which should be executed in some worker.
type Job func()

type Pool struct {
	// job 队列
	JobQueue chan Job
	// 分发器
	dispatcher *dispatcher
	wg         sync.WaitGroup
}

// Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
// 分配具体的 消费者数量 numWokers；分配任务channel的缓存量
func NewPool(numWorkers int, jobQueueLen int) *Pool {
	jobQueue := make(chan Job, jobQueueLen)
	workerPool := make(chan *worker, numWorkers)

	pool := &Pool{
		JobQueue:   jobQueue,
		dispatcher: newDispatcher(workerPool, jobQueue),
	}

	return pool
}

// In case you are using WaitAll fn, you should call this method
// every time your job is done.
//
// If you are not using WaitAll then we assume you have your own way of synchronizing.
func (p *Pool) JobDone() {
	p.wg.Done()
}

// How many jobs we should wait when calling WaitAll.
// It is using WaitGroup Add/Done/Wait
func (p *Pool) WaitCount(count int) {
	p.wg.Add(count)
}

// Will wait for all jobs to finish.
func (p *Pool) WaitAll() {
	p.wg.Wait()
}

// Will release resources used by pool
func (p *Pool) Release() {
	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}
