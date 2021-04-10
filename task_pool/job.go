package task_pool

import "go-common/utils"

type Runnable func(stopChan <-chan bool)

func NewRunnable(task func(stopChan <-chan bool, args ...interface{}), args ...interface{}) Runnable {
	return func(stopChan <-chan bool) {
		task(stopChan, args...)
	}
}

func NewRunnableT(fn interface{}, args ...interface{}) Runnable {
	return func(stopChan <-chan bool) {
		// stopChan 插入到第一个参数中
		_args := append([]interface{}{}, stopChan)
		_args = append(_args, args...)
		utils.Invoke(fn, _args...)
	}
}

type Job struct {
	runnable Runnable
	onDone   func(job *Job)
	running  bool
	complete bool
}

func newJob(runnable Runnable, onDone func(job *Job)) *Job {
	return &Job{
		runnable: runnable,
		onDone:   onDone,
		running:  false,
		complete: false,
	}
}

/**
 * 阻塞运行
 */
func (j *Job) Invoke(stopChan <-chan bool) {

	j.running = true

	defer func() {
		j.running = false
		j.complete = true
		if j.onDone != nil {
			j.onDone(j)
		}
	}()

	j.runnable(stopChan)
}

/**
 * 是否正在运行
 */
func (j *Job) IsRunning() bool {
	return j.running
}

/**
 * 是否运行结束了
 */
func (j *Job) IsComplete() bool {
	return j.complete
}
