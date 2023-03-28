package task

import (
	"container/list"
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type JobDoneHandler func(j Job, err error)

type Task struct {
	jobs        *list.List
	mu          sync.Mutex
	doneHandler JobDoneHandler

	// 有缓冲的队列，保证只会同时运行concurrentCount个任务
	queue chan struct{}
	// 无缓冲，保证run函数中for循环在无数据时阻塞、有数据时触发继续
	hasJobs chan struct{}

	// 所有正在运行的job任务的控制总线
	busCtx    context.Context
	busCancel context.CancelFunc
	// 强制退出
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// NewTask 创建一个任务池，支持任务池服务，和一次性运行池
//  1. 当任务池Stop关闭后，RunXX仍然等待正在运行的任务跑完后退出；Shutdown则会让Run立即推出，但是此时Job任务是否退出不确定。
//
// 2. Job任务从队列中取出后，确保已经开始运行。绝对不会出现在关闭任务池时这种临界点时，这个任务即没有跑，又不在原队列中（比如：Job任务取了放在chan中就会出现这种情况）
func NewTask(concurrentCount int) *Task {
	return &Task{
		jobs:        list.New(),
		mu:          sync.Mutex{},
		queue:       make(chan struct{}, concurrentCount),
		hasJobs:     make(chan struct{}),
		doneHandler: func(j Job, err error) {},

		busCtx:         context.Background(),
		busCancel:      func() {},
		shutdownCtx:    context.Background(),
		shutdownCancel: func() {},
	}
}

func (t *Task) SetJobDoneHandler(fn JobDoneHandler) *Task {
	t.doneHandler = fn
	return t
}

// Submit 添加任务
func (t *Task) Submit(callbacks ...func(ctx context.Context)) *Task {
	t.mu.Lock()
	for _, callback := range callbacks {
		t.submit(Job{Callback: callback, State: Prepare})
	}
	t.mu.Unlock()

	return t
}

// SubmitWithTimeout 添加一个有时间限制的任务，如果任务超时，会调用errorHandler
func (t *Task) SubmitWithTimeout(callback func(ctx context.Context), timeout time.Duration) *Task {
	t.mu.Lock()
	t.submit(Job{Callback: callback, Timeout: timeout, State: Prepare})
	t.mu.Unlock()

	return t
}

func (t *Task) submit(j Job) {
	t.jobs.PushBack(j)

	// 当jobs为空的时候，有新任务就会塞一次数据进去，让run中for不再阻塞
	select {
	case t.hasJobs <- struct{}{}:
	default:
	}

}

// RunOnce 运行jobs中的任务（需添加job）直到所有jobs完成、或 Stop、Ctrl+C
// 不能同时运行RunOnce，RunServe
func (t *Task) RunOnce() {
	t.run(true)
}

// RunServe 持续服务，可随时 Submit Job，注意 RunServe会一直阻塞直到 Stop、Ctrl+C
// 不能同时运行RunOnce，RunServe
func (t *Task) RunServe() {
	t.run(false)
}

func (t *Task) listenStopSignal(ctx context.Context) {
	go func() {
		exitSign := make(chan os.Signal)
		//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
		signal.Notify(exitSign, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer close(exitSign)

		select {
		case <-ctx.Done():
			// 正常退出协程
		case <-exitSign:
			t.Stop()
		}
		signal.Stop(exitSign)
	}()
}

func (t *Task) run(exitOnJobFinish bool) {
	t.busCtx, t.busCancel = context.WithCancel(context.Background())
	t.shutdownCtx, t.shutdownCancel = context.WithCancel(context.Background())
	defer t.busCancel() // 防止泄漏
	defer t.shutdownCancel()
	// ctrl+c
	t.listenStopSignal(t.busCtx)

	wg := &sync.WaitGroup{}
	wg.Add(1) // 先加1，可以在jobs运行过程中不会被因减为0而出现Wait的错误

for1:
	for {
		select {
		case <-t.busCtx.Done(): //全局退出
			break for1
		default:
		}
		t.mu.Lock()
		el := t.jobs.Front()
		t.mu.Unlock()

		if el != nil {
			job := el.Value.(Job)
			select {
			// 控制同时运行的数量
			case t.queue <- struct{}{}:
				wg.Add(1)
				go t.handle(wg, job) // 异步运行
			case <-t.busCtx.Done(): // 全局退出
				break for1
			}

			// 任务取出了 绝对运行（异步）了才从列表里面移除
			// 其它情况会跳出for
			t.mu.Lock()
			t.jobs.Remove(el)
			t.mu.Unlock()

		} else if exitOnJobFinish { // 完成即退出
			break
		} else { // 没有job了，阻塞等待
			select {
			case <-t.busCtx.Done(): //全局退出
				break for1
			case <-t.hasJobs: // 阻塞等待有新的任务进来
			}
		}
	}

	// for 运行完时-1
	wg.Done()

	// 等待任务完成
	wg.Wait()
}

// Stop 停止任务池（需异步调用），但是RunXXX会等待任务运行完毕
// 如果Job任务没有监听ctx完成自行退出，则RunXX永远不会退出。可以尝试使用Shutdown强制退出
func (t *Task) Stop() {
	t.busCancel()
}

// ShutdownNow 停止任务池（需要异步调用）并立即让RunXX退出。
// 鉴于golang的协程的特性，需要Job任务监听ctx完成自行退出，不然Job会继续执行
func (t *Task) ShutdownNow() {
	t.busCancel()
	t.shutdownCancel()
}

// Shutdown 停止任务池（需异步调用）并尝试在waitTimeout时间内等待任务完成。
// 如果任务在waitTimeout时间内都未完成，RunXXX会立即退出。
// 鉴于golang的协程的特性，需要Job任务监听ctx完成自己退出，不然Job会继续执行
//
//	0 表示立即退出RunXX，等效于ShutdownNow
func (t *Task) Shutdown(waitTimeout time.Duration) {
	t.busCancel()

	if waitTimeout > 0 {
		time.AfterFunc(waitTimeout, func() {
			t.shutdownCancel()
		})
	} else {
		t.shutdownCancel()
	}
}

// 归还队列，触发任务完成
func (t *Task) triggerJobDone(wg *sync.WaitGroup, job Job, err error) {
	// 运行完毕则让出队列（保持并发数量的基础）
	<-t.queue
	// 减少wg（run依赖wg来等待所有任务完成）
	wg.Done()
	job.FinishAt = time.Now()

	t.doneHandler(job, err)
}

// 真正执行job的函数，注意：超时后只是归还了队列，Job.Callback如果不监听ctx，则可能还在运行（泄漏）
func (t *Task) handle(wg *sync.WaitGroup, job Job) {
	var jobCtx = t.busCtx
	var jobCancel context.CancelFunc
	var err error

	// 用于超时
	var quiteCh = make(chan struct{})
	defer close(quiteCh)

	go func() {
		// 当quiteCh触发时（正常退出或超时），或者shutdown时，都会立即归还队列，Job.Callback可能还在运行（泄漏）
		select {
		case <-t.shutdownCtx.Done():
		case <-quiteCh:
		}
		t.triggerJobDone(wg, job, err)
	}()

	// 采集错误
	defer func() {
		if err1 := recover(); err1 != nil { // panic
			job.State = Panic
			err = fmt.Errorf("[Task]job exection panic: %v", err1)
		}
	}()

	// 超时
	if job.Timeout > 0 {
		// 新建子超时context，控制job的退出
		jobCtx, jobCancel = context.WithTimeout(t.busCtx, job.Timeout)
		defer jobCancel()
		// timeout后强制让出队列，但是本任务可能会继续运行
		timer := time.AfterFunc(job.Timeout, func() {
			job.State = Timeout
			err = context.DeadlineExceeded
			quiteCh <- struct{}{} // 超时退出
		})
		// 如果任务在规定时间内结束，则停用timer
		defer timer.Stop()
	}

	job.State = Running
	job.RunAt = time.Now()
	// 真正执行函数
	job.Callback(jobCtx)
	job.State = Done
}

func (t *Task) RunningCount() int {
	return len(t.queue)
}

// RemainJobs 剩余没执行完的任务
func (t *Task) RemainJobs() []Job {
	t.mu.Lock()
	defer t.mu.Unlock()

	var jobs []Job
	for el := t.jobs.Front(); el != nil; el = el.Next() {
		jobs = append(jobs, el.Value.(Job))
	}

	return jobs
}
