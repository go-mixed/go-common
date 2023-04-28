package timeUtils

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	var i atomic.Int32
	var times []time.Time
	ticker := NewTicker(100*time.Millisecond, func() {
		times = append(times, time.Now())
		t.Logf("%d - now: %s", i.Add(1), time.Now())
		time.Sleep(1 * time.Second)

	}, 3)

	time.Sleep(5 * time.Second)
	ticker.Stop()

	if len(times) != 15 {
		t.Errorf("times length must be 15")
		t.Fail()
	}

	if delta := times[14].Sub(times[0]); delta >= 500*time.Millisecond {
		t.Errorf("duration delta must be 5s")
		t.Fail()
	} else {
		t.Logf("delta %.4f", delta.Seconds())
	}

	time.Sleep(1 * time.Second)
}
