package task_pool

import (
	"context"
	"go-common/utils/core"
	"runtime"
	"sync"
	"time"
)

type Runnable func(ctx context.Context)

func NewRunnable(task func(ctx context.Context, args ...interface{}), args ...interface{}) Runnable {
	return func(ctx context.Context) {
		task(ctx, args...)
	}
}

func NewRunnableT(fn interface{}, args ...interface{}) Runnable {
	return func(ctx context.Context) {
		// ctx 插入到第一个参数中
		_args := append([]interface{}{}, ctx)
		_args = append(_args, args...)
		core.Invoke(fn, _args...)
	}
}

type Job struct {
	frame          runtime.Frame
	runnable       Runnable
	runningTimeout time.Duration
	ctxCancel      context.CancelFunc
	onDone         func(job *Job)
	running        bool
	// 如果onDone被回调，需要检查complete是否为true，
	// 不为true则考虑
	complete bool
	stopChan chan struct{}
}

func newJob(runnable Runnable, onDone func(job *Job)) *Job {
	return newJobWithTimeout(runnable, onDone, -1)
}

func newJobWithTimeout(runnable Runnable, onDone func(job *Job), timeout time.Duration) *Job {
	frames := 2
	if timeout == -1 { // 说明由newJob调用
		frames = 3
	}
	if timeout < 0 {
		timeout = 0
	}
	return &Job{
		frame:          core.GetFrame(frames),
		runnable:       runnable,
		runningTimeout: timeout,
		onDone:         onDone,
		running:        false,
		complete:       false,
		stopChan:       make(chan struct{}),
	}
}

// Invoke 阻塞运行, 等待自然结束, 或者timeout结束
// 注意: timeout之后会触发ctx.Done
// 如果该任务在shutdownTimeout之后仍然无法退出, 会强制执行completeJob,并从正在运行任务中移除
// 此时该任务仍会继续运行, 并导致泄露
func (j *Job) Invoke(ctx context.Context, shutdownTimeout time.Duration) {

	j.running = true
	var forceQuitTimer *time.Timer // 强制退出的timer
	var subCtx context.Context
	if j.runningTimeout > 0 {
		// runningTimeout 之后发送停止指令
		subCtx, j.ctxCancel = context.WithTimeout(ctx, j.runningTimeout)
		// runningTimeout + shutdownTimeout 之后发强制停止指令
		forceQuitTimer = time.NewTimer(j.runningTimeout + shutdownTimeout)
	} else {
		subCtx, j.ctxCancel = context.WithCancel(ctx)
	}

	// 运行完毕, 执行j.completeJob()
	go func() {
		defer j.completeJob(false)
		j.runnable(subCtx)
		// 如果是正常运行结束, 则关闭 forceQuitTimer
		if forceQuitTimer != nil {
			forceQuitTimer.Stop()
		}
	}()

	// 阻塞等待任务完成 或者
	if forceQuitTimer != nil {
		// 同时监听timer, 或stopChan, 谁先触发 先退出本函数
		select {
		case <-j.stopChan:
		case <-forceQuitTimer.C:
			j.completeJob(true) // 如果是forceQuitTimer触发, 也会强制退出
		}
	} else {
		// 没有timer时, 等待停止通道
		select {
		case <-j.stopChan:
		}
	}

}

var jobMutex = sync.Mutex{}

func (j *Job) completeJob(isForceQuit bool) {
	jobMutex.Lock()
	defer jobMutex.Unlock()
	// 一个任务只会执行一次
	if j.running {
		j.running = false
		j.complete = !isForceQuit
		if j.ctxCancel != nil {
			j.ctxCancel()
		}
		j.ctxCancel = nil
		if j.onDone != nil {
			j.onDone(j)
		}
		j.onDone = nil
		close(j.stopChan)
	}
}

// IsRunning 是否正在运行
func (j *Job) IsRunning() bool {
	return j.running
}

// IsComplete 是否正常运行结束了， 通过这个可以判断是否是timeout强制终止的任务
func (j *Job) IsComplete() bool {
	return j.complete
}
