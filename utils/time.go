package utils

import "time"

func TimeStampToTime(timestamp int64) {
	time.Unix(timestamp, 0)
}

func TimeToStandard(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func TimeToIso8601(t time.Time) string {
	// .Format("2006-01-02T15:04:05-0700")
	return t.Format(time.RFC3339)
}

func Iso8601ToTime(str string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		t, err = time.Parse(time.RFC3339, str)
	}

	if err == nil {
		return t
	}
	return time.Time{}
}
