package task

import (
	"context"
	"github.com/pkg/errors"
	"time"
)

type worker struct {
	task *Task
	// id 是从1开始
	id int
}

func createWorker(task *Task) *worker {
	w := &worker{task: task, id: -1}
	return w
}

func (w *worker) run(ctx context.Context) {
	defer w.recycleWorker() // 将任务回收到pool中

	for {
		select {
		case <-ctx.Done(): // worker退出
			w.task.logger.Debugf("[Worker-%d]quited cause of context", w.id)
			break
		default:
		}

		// 使用锁，取出头部的Job
		w.task.mu.Lock()
		// 如果workerCount做了调整，当前id比maxWorkerCount大时，就退出
		if w.id > w.task.maxWorkerCount {
			w.task.mu.Unlock()
			w.task.logger.Debugf("[Worker-%d]quited cause of max-worker-count", w.id)
			break
		}
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
	var quited = false
	defer func() {
		// 采集错误
		if err := recover(); err != nil { // panic
			job.State = Panic
			job.Error = errors.Errorf("[Worker-%d]job exection panic: %v", w.id, err)
		} else if job.State == Running {
			job.State = Done
		}
		if !quited {
			w.triggerJobDone(job)
		}
	}()

	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()

	// 已配置超时参数
	if job.Timeout > 0 {
		// timeout后强制让出队列
		timer := time.AfterFunc(job.Timeout, func() {
			job.State = Timeout
			job.Error = context.DeadlineExceeded
			jobCancel()
			w.task.logger.Debugf("[Worker-%d]job execution timeout %dms", w.id, job.Timeout.Milliseconds())
			quited = true
			w.triggerJobDone(job)
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
	w.id = -1
	w.task.pool.Put(w)
	w.task.triggerAction(workerQuitAction)
}

func (w *worker) SetID(id int) {
	w.id = id
}
