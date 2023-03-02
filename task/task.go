package task

import (
	"container/list"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Job struct {
	callback func(ctx context.Context)
	timeout  time.Duration
}

type ErrorHandlerFunc func(j Job, err error)

type Task struct {
	jobs         list.List
	mu           sync.Mutex
	errorHandler ErrorHandlerFunc

	queue    chan struct{}
	hasJobs  chan struct{}
	shutdown bool

	ctx    context.Context
	cancel context.CancelFunc
}

func NewTask(concurrentCount int) *Task {
	return &Task{
		jobs:         list.List{},
		mu:           sync.Mutex{},
		queue:        make(chan struct{}, concurrentCount),
		hasJobs:      make(chan struct{}),
		shutdown:     false,
		errorHandler: func(j Job, err error) {},
		ctx:          context.Background(),
		cancel:       func() {},
	}
}

func (t *Task) SetErrorHandler(fn ErrorHandlerFunc) *Task {
	t.errorHandler = fn
	return t
}

// Submit 添加任务
func (t *Task) Submit(callbacks ...func(ctx context.Context)) *Task {
	t.mu.Lock()
	for _, callback := range callbacks {
		t.submit(Job{callback: callback})
	}
	t.mu.Unlock()

	return t
}

// SubmitWithTimeout 添加一个有时间限制的任务，如果任务超时，会调用errorHandler
func (t *Task) SubmitWithTimeout(callback func(ctx context.Context), timeout time.Duration) *Task {
	t.mu.Lock()
	t.submit(Job{callback, timeout})
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
	t.shutdown = false
	t.ctx, t.cancel = context.WithCancel(context.Background())
	// ctrl+c
	t.listenStopSignal(t.ctx)

	wg := &sync.WaitGroup{}
	wg.Add(1) // 先加1，可以在jobs运行过程中不会被因减为0而出现Wait的错误

for1:
	for {
		select {
		case <-t.ctx.Done(): //全局退出
			break for1
		default:
		}
		t.mu.Lock()
		el := t.jobs.Front()
		t.mu.Unlock()

		if el != nil {
			job := el.Value.(Job)
			select {
			case t.queue <- struct{}{}: // 控制同时运行的数量
				wg.Add(1)
				go t.handle(wg, job) // 异步运行
			case <-t.ctx.Done(): // 全局退出
				break for1
			}

			t.mu.Lock()
			t.jobs.Remove(el)
			t.mu.Unlock()

		} else if exitOnJobFinish {
			break
		} else {
			select {
			case <-t.ctx.Done(): //全局退出
				break for1
			case <-t.hasJobs: // 阻塞等待有新的任务进来
			}
		}
	}

	// for 运行完时-1
	wg.Done()

	if !t.shutdown {
		wg.Wait()
	}
}

// Stop 停止任务池（需异步调用），但是RunXXX会等待任务运行完毕
func (t *Task) Stop() {
	t.cancel()
}

// Shutdown 停止任务池（需异步调用），RunXXX便会立即退出
func (t *Task) Shutdown() {
	t.shutdown = true
	t.cancel()
}

// 真正执行job的函数
func (t *Task) handle(wg *sync.WaitGroup, job Job) {
	var jobCtx context.Context = t.ctx
	var jobCancel context.CancelFunc

	defer func() {
		// 运行完毕则让出队列（保持并发数量的基础）
		<-t.queue
		// 减少wg（run依赖wg来等待所有任务完成）
		wg.Done()
	}()

	defer func() {
		if err := jobCtx.Err(); err != nil && !errors.Is(err, context.Canceled) {
			t.errorHandler(job, err)
		} else if err1 := recover(); err1 != nil {
			t.errorHandler(job, fmt.Errorf("[Task]job exection error: %v", err1))
		}
	}()

	if job.timeout > 0 {
		jobCtx, jobCancel = context.WithTimeout(t.ctx, job.timeout)
		defer jobCancel()
	}

	job.callback(jobCtx)
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
