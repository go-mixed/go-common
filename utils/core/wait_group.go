package core

import (
	"go.uber.org/atomic"
	"sync"
)

// WaitGroup 一个不会因为 Done 时负数而导致panic的WaitGroup
type WaitGroup struct {
	sync.WaitGroup
	counter atomic.Int64
}

// Add 注意: 如果传递一个超出WaitGroup的负数, 会panic
func (w *WaitGroup) Add(delta int) {
	w.counter.Add(int64(delta))
	w.WaitGroup.Add(delta)
}

// Done 判别并阻止负数panic
func (w *WaitGroup) Done() bool {
	if w.counter.Dec() >= 0 {
		w.WaitGroup.Done()
		return true
	} else {
		w.counter.Store(0)
		return false
	}
}

// Counter 返回计数器
func (w *WaitGroup) Counter() int64 {
	return w.counter.Load()
}

// Clear 清楚计数器
func (w *WaitGroup) Clear() {
	var i int64
	for i = w.counter.Load(); i >= 0; i-- {
		w.Done()
	}
}
