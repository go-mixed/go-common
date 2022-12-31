package task_pool

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/list"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Params struct {
	Name            string                         // 名称
	NumWorkers      int                            // 同时运行的任务
	ShutdownTimeout time.Duration                  // shutdown的超时时间 ms
	ShutdownHandler func(running int, stopped int) // shutdown 退出时候的回调
}

func (p *Params) validate() error {
	if p.NumWorkers <= 0 {
		return errors.New(fmt.Sprintf("[%s]executor params: non positive NumWorkers", p.Name))
	}
	return nil
}

// DefaultExecutorParams 默认 Executor 参数
//
//	NumWorkers: 默认CPU核心数
//	MaxJobQueueCapacity: 1000 最大等待的任务数为1000
//	MaxJobQueueWaitTime: 30s 设置0表示不过期
//	ShutdownTimeout: 3 seconds 发送Stop指令后最多等待多少秒退出
func DefaultExecutorParams() Params {
	return Params{
		Name:            core.GetFrame(1).Function,
		NumWorkers:      runtime.NumCPU(),
		ShutdownTimeout: 3 * time.Second,
	}
}

func NewExecutorParams(NumWorkers int, ShutdownTimeout time.Duration, Name string) Params {
	if Name == "" {
		Name = core.GetFrame(1).Function
	}
	return Params{
		Name:            Name,
		NumWorkers:      NumWorkers,
		ShutdownTimeout: ShutdownTimeout,
	}
}

type Executor struct {
	wg      *core.WaitGroup // 一个可以Clear的WaitGroup
	mu      *sync.Mutex
	params  *Params // 参数
	stopped bool    // 任务池是否已经停止了

	rootContext context.Context // 全局停止Context
	rootCancel  context.CancelFunc

	termChan chan os.Signal

	runningJobs *list_utils.ConcurrencyList
	queueJobs   *list_utils.ConcurrencyList

	logger logger.ILogger
}

// NewExecutor 创建一个任务运行池，只能同时运行limit个任务
func NewExecutor(params Params, logger logger.ILogger) (*Executor, error) {
	return NewExecutorContext(context.Background(), params, logger)
}

// NewExecutorContext 根据ctx创建一个任务运行池, 当ctx被cancel时, 所有任务都会被停止
func NewExecutorContext(ctx context.Context, params Params, logger logger.ILogger) (*Executor, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	stopCtx, cancel := context.WithCancel(ctx)

	executor := &Executor{
		mu:          &sync.Mutex{},
		params:      &params,
		stopped:     false,
		wg:          &core.WaitGroup{},
		rootContext: stopCtx,
		rootCancel:  cancel,

		queueJobs:   list_utils.NewConcurrencyList(),
		runningJobs: list_utils.NewConcurrencyList(),
		logger:      logger,
	}

	return executor, nil
}

// Submit 添加一个或多个任务，添加之后，如果任务池已经满，会排队等待执行，不然会立即执行
//
// 注意: 本函数只会添加任务, 不会运行任务, 所以不会阻塞。
// 对于持久的任务，一定要监听 ctx.Done 通道后退出任务，不然在Ctrl+C时导致程序无法正确的退出。
//
// 请参考下面例子完成持久任务的退出操作：
//
//	e.Submit(func(ctx context.Context) error {
//		// 死循环，说明这是一个持久的任务
//		for {
//			select {
//			case <-ctx.Done(): // 监听通道, 做好随时退出的准备
//				print("exit task")
//				return
//			default: // 没有收到信息时，会正常往下执行
//			}
//			... do something
//		}
//	}
func (e *Executor) Submit(runnables ...Runnable) []*Job {
	res := make([]*Job, 0, len(runnables))

	for _, runnable := range runnables {
		job := newJob(runnable, e.onJobDone)
		res = append(res, job)
		e.queueJobs.Push(job)
	}

	e.runNextJob()

	return res
}

// SubmitWithTimeout 提交一个带超时时间的任务
//
// 任务最长运行时间为timeout，超过这个时间时，会执行cancel并触发ctx.Done()
// 如果任务不响应ctx.Done()，那会在正在运行任务剔除，并造成泄露
func (e *Executor) SubmitWithTimeout(runnable Runnable, timeout time.Duration) *Job {
	job := newJobWithTimeout(runnable, e.onJobDone, timeout)
	e.queueJobs.Push(job)
	e.runNextJob()
	return job
}

// Stop 停止所有任务，会阻塞等待正在运行的任务运行完毕，或者在ShutdownTimeout超时后强制退出本函数
//
// 正在运行的任务，会关闭 rootContext 通道通知它停止。
// 如果某任务在ShutdownTimeout之后仍未停止，Executor 会清理正在运行的任务[列表]，然后退出Stop。注意：如果该任务仍然无法自行结束，会导致泄露
// 对于未运行的任务，不会删除，后续可以通过 QueueJobs 查看
// 注意：停止后的任务池将结束生命周期，即使再添加任务到 Submit 中也不会启动，除非 Reset
func (e *Executor) Stop() {
	e.StopWithTimeout(e.params.ShutdownTimeout)
}

// StopAfter 在timeout之后下达停止指令
func (e *Executor) StopAfter(timeout time.Duration) {
	time.AfterFunc(timeout, e.Stop)
}

// StopWithTimeout 发送rootContext关闭通道, 如果timeout之后仍然有任务在运行, 会强制退出StopWithTimeout, 具体解释参见 Stop
func (e *Executor) StopWithTimeout(timeout time.Duration) {
	e.mu.Lock() // 此任务只能被1个协程运行

	// 重复执行
	if e.stopped {
		e.mu.Unlock()
		return
	}

	e.stopped = true
	e.rootCancel() // 发送cancel指令到正在执行的任务中
	e.mu.Unlock()

	var needStoppingJobCount = e.runningJobs.Len()
	var stillRunningJobCount int

	// 超过 ShutdownTimeout 直接退出
	go func() {
		// Graceful shutdown
		afterC := time.After(timeout)
		ticker := time.NewTicker(timeout / 10)
		defer ticker.Stop() // 防止泄露
		for {
			// 运行的任务已经结束, 则自动退出
			if e.runningJobs.Len() == 0 {
				break
			}

			select {
			case <-afterC: // 如果超时仍然有任务在运行
				stillRunningJobCount = e.runningJobs.Len()

				// 清理wg，让下方e.Wait解除阻塞
				// 正在运行的任务将不可控，如果某个任务无法自行停止，会导致泄露
				e.wg.Clear()

				e.logger.Infof("[%s]stop incorrect, %d job(s) still running.", e.params.Name, stillRunningJobCount)
				return // 退出协程
			case <-ticker.C: // 每timeout/10检查下任务是否运行完毕
			}
		}
	}()

	// 阻塞 直到所有的任务停止
	e.Wait()

	if e.params.ShutdownHandler != nil {
		e.params.ShutdownHandler(stillRunningJobCount, needStoppingJobCount-stillRunningJobCount)
	}

	e.logger.Infof("[%s]all running jobs stopped.", e.params.Name)
}

// Reset 对于已经停止的任务池，可以再次启动
// keepRemainTasks bool: 是否保留剩余的任务，如果任务池中还有剩余的任务，在执行 Reset 之后会立即启动
func (e *Executor) Reset(keepRemainJobs bool) {
	e.ResetContext(context.Background(), keepRemainJobs)
}

func (e *Executor) ResetContext(ctx context.Context, keepRemainJobs bool) {

	if !keepRemainJobs {
		e.queueJobs.Clear()
	}

	e.mu.Lock() // 此任务只能被1个协程运行

	if e.stopped {
		e.stopped = true
		e.termChan = nil
		e.rootContext, e.rootCancel = context.WithCancel(ctx)
		e.runningJobs.Clear()
	}

	e.mu.Unlock()

	e.runNextJob()
}

// 往任务池中塞任务并执行，如果任务池已满，则不会做任何操作。
// 此方法会被 Submit onJobDone 触发
func (e *Executor) runNextJob() {
	e.mu.Lock() // 此任务只能被1个协程运行
	defer e.mu.Unlock()

	if e.stopped {
		return
	}

	delta := e.params.NumWorkers - e.runningJobs.Len()

	for i := 0; i < delta; i++ {
		job := e.queueJobs.Pop()
		if job != nil {
			e.invokeJob(job.(*Job)) // 异步执行task
		} else {
			break // 没有job
		}
	}

}

// 异步执行task
func (e *Executor) invokeJob(job *Job) {
	e.wg.Add(1)
	e.runningJobs.Push(job)
	// 非阻塞运行
	go job.Invoke(e.rootContext, e.params.ShutdownTimeout)
}

// 任务结束时候的回调
// 会触发 runNextJob 让空闲任务池获得任务
func (e *Executor) onJobDone(job *Job) {
	// 在正在运行的任务中删除此job
	if !job.IsComplete() {
		e.logger.Warnf("task force quit with timeout: %.3fs, job submitted by \"%s\" in %s:%d ", job.runningTimeout.Seconds(), job.frame.Function, job.frame.File, job.frame.Line)
	}
	e.runningJobs.Remove(job)

	e.runNextJob() // 执行下一个任务

	e.wg.Done() // 必须在新任务已经派发后才能Done，也runNextJob之后，不然会导致Wait过早退出
}

// IsRunning 是否有任务正在运行
func (e *Executor) IsRunning() bool {
	return e.runningJobs.Len() > 0
}

// QueueJobs 未执行的任务，即待执行的任务
func (e *Executor) QueueJobs() []Runnable {
	e.mu.Lock() // 此任务只能被1个协程运行
	defer e.mu.Unlock()

	if e.queueJobs.Len() <= 0 {
		return nil
	}

	runnables := make([]Runnable, 0, e.queueJobs.Len())

	for e := e.queueJobs.HeadElement(); e != nil; e = e.Next() {
		runnables = append(runnables, e.Value.(*Job).runnable)
	}

	return runnables
}

// RunningJobs 正在运行的任务
func (e *Executor) RunningJobs() []Runnable {
	if e.runningJobs.Len() <= 0 {
		return nil
	}

	runnables := make([]Runnable, 0, e.runningJobs.Len())

	for e := e.runningJobs.HeadElement(); e != nil; e = e.Next() {
		runnables = append(runnables, e.Value.(*Job).runnable)
	}

	return runnables
}

// ListenStopSignal 监听Ctrl+C/kill的退出消息
// 在主线程使用 Executor 时，最好调用本方法来达到Ctrl+C退出，不然会出现退出异常的情况
// 如果需要监听一个外部的 stopChan, Context, 使用 ListenStopChan / ListenContext
func (e *Executor) ListenStopSignal() {
	e.mu.Lock()
	defer e.mu.Unlock()
	// 只会执行一次
	if e.termChan == nil {
		e.termChan = make(chan os.Signal)
		//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
		signal.Notify(e.termChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			select {
			case s := <-e.termChan:
				signal.Stop(e.termChan) // remove signal
				e.logger.Infof("[%s]get signal %s, quiting.", e.params.Name, s)
				e.Stop()
				return
			case <-e.rootContext.Done(): // 为了避免本协程一直阻塞，当e.rootContext关闭时，会退出本协程
				signal.Stop(e.termChan) // remove signal
				return
			}
		}()
	}
}

// ListenStopChan 监听一个外部的outerStopChan
// 当outerStopChan有值或关闭时，会触发 Stop。
// 本函数类似于 ListenStopSignal，只是前者监听的是Ctrl+C，而本函数监听的是outerStopChan
func (e *Executor) ListenStopChan(outerStopChan <-chan struct{}) {
	go func() {
		select {
		case <-outerStopChan:
			e.logger.Infof("[%s]quit by outer stop chan.", e.params.Name)
			e.Stop()
		case <-e.rootContext.Done(): // 为了避免本协程一直被阻塞，当 e.rootContext 关闭时，也会退出本协程

		}
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.termChan != nil {
			signal.Stop(e.termChan) // remove signal
		}
	}()
}

// ListenContext 监听一个外部的 Context, 这个 Context 可以和 NewExecutorContext 中传递的无关
// 当Context.Done时，会触发 Stop。
func (e *Executor) ListenContext(ctx context.Context) {
	e.ListenStopChan(ctx.Done())
}

// Wait 阻塞等待所有任务运行完毕
func (e *Executor) Wait() {
	e.wg.Wait()
}
