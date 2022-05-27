package core

import (
	"go.uber.org/atomic"
	"sync"
)

// WaitGroup 一个不会因为 Done 时负数而导致panic的WaitGroup
// 注意：原 WaitGroup 代码的Add方法中为减少atomic的运用，专门做了性能优化，
// 而本文件违背了这个初衷，增加了一次atomic操作。如果没有会到负数的场景，尽量使用原 WaitGroup
type WaitGroup struct {
	sync.WaitGroup
	counter atomic.Int32
}

// Add 注意: 如果传递一个超出WaitGroup的负数, 仍然会panic
func (w *WaitGroup) Add(delta int) {
	w.counter.Add(int32(delta))
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
func (w *WaitGroup) Counter() int32 {
	return w.counter.Load()
}

// Clear 清除所有计数
func (w *WaitGroup) Clear() {
	var i int32
	for i = w.counter.Load(); i >= 0; i-- {
		w.Done()
	}
}
