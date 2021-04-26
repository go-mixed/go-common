package time

import "time"

type clock struct {
	lastAt   time.Time
	recorder []time.Time
}

type ClockIn struct {
	clocks map[string]*clock
}

// NewClockIn 新建打卡机
func NewClockIn() *ClockIn {
	return &ClockIn{clocks: map[string]*clock{}}
}

// Reset 重置打卡机
func (c *ClockIn) Reset(name string) {
	if _, ok := c.clocks[name]; ok {
		c.clocks[name] = nil
	}
}

// ResetAll 重置所有打卡机
func (c *ClockIn) ResetAll() {
	for k := range c.clocks {
		c.Reset(k)
	}
}

// IsAfter 最后一次打卡是否已经超过了duration
func (c *ClockIn) IsAfter(name string, duration time.Duration) bool {
	return c.Duration(name) > duration
}

// Tick 打卡一次
func (c *ClockIn) Tick(name string) {
	var _clock *clock
	var ok bool
	if _clock, ok = c.clocks[name]; ok && _clock != nil {
		_clock.lastAt = time.Now()
	} else {
		_clock = &clock{
			lastAt: time.Now(),
		}
		c.clocks[name] = _clock
	}
	_clock.recorder = append(_clock.recorder, _clock.lastAt)
}

// LastTickAt 最后一次打卡的时间
func (c *ClockIn) LastTickAt(name string) time.Time {
	if v, ok := c.clocks[name]; ok && v != nil {
		return v.lastAt
	}
	return time.Time{}
}

// Duration 最后一次打卡距离现在的时长
func (c *ClockIn) Duration(name string) time.Duration {
	return time.Now().Sub(c.LastTickAt(name))
}

// DoIfAfter 如果最后一次打开已经超过了duration，则执行 fn，并打卡一次，否则将立即退出（不会执行fn），并返回 false
// 注意：这并不是一个time.After方法
// 符合条件是指：duration < time.Now() - LastTickAt
func (c *ClockIn) DoIfAfter(name string, duration time.Duration, fn func()) bool {
	if c.IsAfter(name, duration) {
		fn()
		c.Tick(name)
		return true
	}
	return false
}

// DoAfter 如果符合条件，会立即执行fn，不然，会使用time.AfterFunc方法等待符合条件后执行，最终都会打卡一次
// 符合条件是指：duration < time.Now() - LastTickAt
func (c *ClockIn) DoAfter(name string, duration time.Duration, fn func()) {
	lastAt := c.LastTickAt(name)
	delta := time.Now().Sub(lastAt)
	if duration <= delta { // 已经超过了要求的时间
		fn()
		c.Tick(name)
	} else {
		time.AfterFunc(duration-delta, func() {
			c.DoAfter(name, duration, fn)
		})
	}
}
