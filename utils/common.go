package utils

import (
	"reflect"
	"runtime"
)

// If 类似三目运算, 但是这不是真正的三目运算, 因为不论 e为何值, a, b的表达式都会被运算, 其它语言中, e为true时, b不会运算
// 比如: If(a != nil, a.XX, "default"), 如果a为nil, a.XX运算会导致程序会崩溃
// 比如: If(e, a.fastFn(), a.SlowFn()), 不论e为何值, fastFn/SlowFn 都会被运行, 只是不返回SlowFn的值罢了
// 此时只能 if a != nil {} else {}
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

// GetFrame 获取调用栈列表
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

// IsInterfaceNil 指针是否为nil
func IsInterfaceNil(v interface{}) bool {
	vOf := reflect.ValueOf(v)
	return v == nil || (vOf.Kind() == reflect.Ptr && vOf.IsNil())
}
