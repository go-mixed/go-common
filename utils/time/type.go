package time_utils

import "time"

type MillisecondDuration int64

func (md MillisecondDuration) ToDuration() time.Duration {
	return time.Duration(md) * time.Millisecond
}
