package timeUtils

import (
	"fmt"
	"time"
)

func DurationToString(t time.Duration) string {
	h := int64(t / time.Hour)
	m := int64(t/time.Minute) - h*60

	s := t.Seconds() - float64(h*3600+m*60)
	precision := s - float64(int64(s)) // 小数位

	if precision < 0.001 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, int64(s)) // 没小数
	} else {
		return fmt.Sprintf("%02d:%02d:%06.3f", h, m, s)
	}

}
