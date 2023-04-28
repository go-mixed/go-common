package task

import (
	"container/list"
	"context"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type JobDoneHandler func(j *Job)

type action uint8

const (
	workerStartAction action = iota
	workerQuitAction
	stoppingAction
	quitAfterDoneAction
	quitImmediatelyAction
)

type Task struct {
	pool        sync.Pool
	mu          sync.Mutex
	doneHandler JobDoneHandler

	jobs *list.List
	// 触发器，触发一次动整个Task动一下，无缓存
	actions chan action

	runningCount   atomic.Int32
	maxWorkerCount int
	logger         utils.ILogger
}

// NewTask 创建一个任务池，支持任务池服务，和一次性运行池
//  1. 两种关闭方式：
//     - Stop：RunXX仍然等待正在运行的Job任务运行结束后退出；
//     - Shutdown会让RunXX立即退出，但是此时Job任务是否退出是不确定的。
//
// 2. Job任务从队列中取出后，本程序会确保已经开始运行。绝对不会出现在关闭任务池时这种临界点时，任务即没有跑，又不在原队列中（比如：Job任务取了放在chan中就会出现这种情况）
func NewTask(maxWorkerCount int) *Task {
	t := &Task{
		pool:           sync.Pool{},
		jobs:           list.New(),
		actions:        make(chan action, maxWorkerCount),
		mu:             sync.Mutex{},
		maxWorkerCount: maxWorkerCount,
		doneHandler:    func(j *Job) {},
		logger:         utils.NewDefaultLogger(),
	}
	t.pool.New = func() any {
		return createWorker(t)
	}
	return t
}

func (t *Task) SetMaxWorkerCount(maxWorkerCount int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.maxWorkerCount = maxWorkerCount
}

func (t *Task) SetLogger(logger utils.ILogger) {
	t.logger = logger
}

func (t *Task) SetJobDoneHandler(fn JobDoneHandler) *Task {
	t.doneHandler = fn
	return t
}

// Submit 添加任务
func (t *Task) Submit(callbacks ...func(ctx context.Context)) *Task {
	t.mu.Lock()
	for _, callback := range callbacks {
		t.submit(&Job{Callback: callback, State: Prepare})
	}
	t.mu.Unlock()

	t.tryTriggerAction(workerStartAction)
	return t
}

// SubmitWithTimeout 添加一个有时间限制的任务，如果任务超时，会调用errorHandler
func (t *Task) SubmitWithTimeout(callback func(ctx context.Context), timeout time.Duration) *Task {
	t.mu.Lock()
	t.submit(&Job{Callback: callback, Timeout: timeout, State: Prepare})
	t.mu.Unlock()

	t.tryTriggerAction(workerStartAction)
	return t
}

func (t *Task) submit(j *Job) {
	t.jobs.PushBack(j)
}

func (t *Task) triggerAction(action action) {
	t.actions <- action
}

func (t *Task) tryTriggerAction(action action) {
	select {
	case t.actions <- action:
	default:

	}
}

// RunOnce 运行jobs中的任务（需先添加job）直到所有jobs完成、或主动 Stop、Ctrl+C
// 不能同时运行RunOnce，RunServe
func (t *Task) RunOnce() {
	// 没有任务
	if t.jobs.Len() <= 0 {
		return
	}
	t.run(true)
}

// RunServe 持续服务，可随时 Submit Job，注意 RunServe会一直阻塞直到主动 Stop、Ctrl+C
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

// 真正运行的方法
func (t *Task) run(exitOnFinish bool) {
	stoppingCtx, stoppingCtxCancel := context.WithCancel(context.Background())
	defer stoppingCtxCancel()
	// ctrl+c
	t.listenStopSignal(stoppingCtx)

	wg := &sync.WaitGroup{}
	wg.Add(1)

for1:
	for act := range t.actions {
		switch act {
		case stoppingAction: // 仅仅调用stoppingCtx
			stoppingCtxCancel()
		case quitAfterDoneAction: // 等待当前任务运行结束之后退出
			break for1
		case quitImmediatelyAction: // 不等待当前任务运行结束，立即退出
			return
		case workerQuitAction:
			wg.Done()                                        // worker wg -1
			if t.runningCount.Add(-1) <= 0 && exitOnFinish { // 运行完毕就结束
				break for1
			}
		case workerStartAction:
			if int(t.runningCount.Load()) < t.maxWorkerCount {
				w := t.pool.Get().(*worker)
				wg.Add(1)

				// worker wg +1
				workerID := t.runningCount.Add(1) // running +1
				w.SetID(int(workerID))
				go w.run(stoppingCtx)
			}
		}
	}

	wg.Done()
	wg.Wait()
}

// Stop 停止任务池（需异步调用），但是RunXX仍然会等待任务运行结束
// 如果Job任务没有监听ctx完成自行退出，则RunXX永远不会退出。可以尝试使用Shutdown强制退出
func (t *Task) Stop() {
	t.triggerAction(stoppingAction)
	t.triggerAction(quitAfterDoneAction)
}

// ShutdownNow 停止任务池（需要异步调用）并立即让RunXX退出。
// 鉴于golang的协程的特性，需要Job任务监听ctx完成自行退出，不然Job会继续执行
func (t *Task) ShutdownNow() {
	t.Shutdown(0)
}

// Shutdown 停止任务池（需异步调用）并尝试在waitTimeout时间内等待任务完成。
// 如果任务在waitTimeout时间内都未完成，RunXXX会立即退出。
// 鉴于golang的协程的特性，需要Job.Callback需监听ctx并退出，不然Job会在脱离管控下继续执行
//
//	0 表示立即退出RunXX，等效于ShutdownNow
func (t *Task) Shutdown(waitTimeout time.Duration) {
	t.triggerAction(stoppingAction)
	if waitTimeout > 0 {
		time.AfterFunc(waitTimeout, func() {
			t.triggerAction(quitImmediatelyAction)
		})
	} else {
		t.triggerAction(quitImmediatelyAction)
	}
}

func (t *Task) RunningCount() int {
	return int(t.runningCount.Load())
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
