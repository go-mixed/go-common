package time_utils

import "time"

type Ticker struct {
	*time.Ticker

	callback func()
	stopCh   chan struct{}
}

func NewTicker(d time.Duration, callback func()) *Ticker {
	t := &Ticker{
		Ticker:   time.NewTicker(d),
		callback: callback,

		stopCh: make(chan struct{}),
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
			t.callback()
		}
	}
}

func (t *Ticker) Stop() {
	t.Ticker.Stop()
	t.stopCh <- struct{}{}
}
