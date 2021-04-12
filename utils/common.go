package utils

import (
	"crypto/md5"
	"encoding/hex"
	"reflect"
	"runtime"
)

func If(e bool, a, b interface{}) interface{} {
	if e {
		if reflect.TypeOf(a).Kind() == reflect.Func {
			return a.(func() interface{})()
		}
		return a
	}
	if reflect.TypeOf(b).Kind() == reflect.Func {
		return b.(func() interface{})()
	}
	return b
}

func Md5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GetFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}
