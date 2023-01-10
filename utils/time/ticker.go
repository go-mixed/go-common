package time_utils

import "time"

type Ticker struct {
	*time.Ticker

	callback func()
	stopCh   chan struct{}
	// 同时运行的任务，如果超过正maxConcurrentRunningCount，则等待下个周期
	runningCh chan struct{}
}

func NewTicker(interval time.Duration, callback func(), maxConcurrentRunningCount int) *Ticker {
	if maxConcurrentRunningCount <= 0 {
		maxConcurrentRunningCount = 1
	}
	t := &Ticker{
		Ticker:   time.NewTicker(interval),
		callback: callback,

		stopCh:    make(chan struct{}),
		runningCh: make(chan struct{}, maxConcurrentRunningCount),
	}

	go t.run()
	return t
}

func (t *Ticker) run() {
	for {
		select {
		case <-t.stopCh:
			break
		case <-t.Ticker.C:
			select {
			case t.runningCh <- struct{}{}: // 能插入任务
				go t.handle() // 运行任务
			default: // 丢弃任务

			}
		}
	}
}

func (t *Ticker) handle() {
	defer func() { // 运行完毕后腾出空位
		<-t.runningCh
	}()
	t.callback()
}

func (t *Ticker) Stop() {
	t.Ticker.Stop()
	t.stopCh <- struct{}{}
}
