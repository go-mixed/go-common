package task

import (
	"context"
	"github.com/pkg/errors"
	"time"
)

type worker struct {
	task *Task
}

func createWorker(task *Task) *worker {
	w := &worker{task: task}
	return w
}

func (w *worker) run(ctx context.Context) {
	defer w.recycleWorker() // 将任务回收到pool中

	for {
		select {
		case <-ctx.Done(): // worker退出
			break
		default:
		}

		// 使用锁，取出头部的Job
		w.task.mu.Lock()
		el := w.task.jobs.Front()
		if el != nil {
			w.task.jobs.Remove(el)
		}
		w.task.mu.Unlock()

		if el != nil {
			w.handle(ctx, el.Value.(*Job))
		} else {
			break // 运行完毕则结束
		}
	}
}

// 归还队列，触发任务完成
func (w *worker) triggerJobDone(job *Job) {
	job.FinishAt = time.Now()
	w.task.doneHandler(job)
}

// 真正执行job的函数，注意：超时后只是归还了队列，Job.Callback如果不监听ctx.Done()，Job会在脱离管控下继续执行
func (w *worker) handle(ctx context.Context, job *Job) {
	defer func() {
		// 采集错误
		if err := recover(); err != nil { // panic
			if job.State == Running {
				job.State = Panic
			}
			job.Error = errors.Errorf("[Task]job exection panic: %v", err)
		} else if job.State == Running {
			job.State = Done
		}
		w.triggerJobDone(job)
	}()

	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()

	// 已配置超时参数
	if job.Timeout > 0 {
		// timeout后强制让出队列
		timer := time.AfterFunc(job.Timeout, func() {
			job.State = Timeout
			jobCancel()
			panic(context.DeadlineExceeded)
		})
		// 如果任务在规定时间内结束，则停用timer
		defer timer.Stop()
	}

	job.State = Running
	job.RunAt = time.Now()
	// 真正执行函数
	job.Callback(jobCtx)
}

func (w *worker) recycleWorker() {
	w.task.pool.Put(w)
	w.task.triggerAction(workerQuitAction)
}
