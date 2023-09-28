package core

import (
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils/conv"
	"reflect"
	"runtime"
)

// If 类似三目运算。
// 但是这不是真正的三目运算, 因为不论 e为何值, a, b的表达式都会被运算, 其它语言中, e为true时, b不会运算
//
//	比如: If(a != nil, a.XX, "default"), 如果a为nil, a.XX运算会导致程序崩溃
//	比如: If(e, a.fastFn(), a.SlowFn()), 不论e为何值, fastFn/SlowFn 都会被运行, 只是不返回SlowFn的值罢了
//
// 上面情况，可以使用 IfT 来避免
func If[T any](e bool, a, b T) T {
	if e {
		return a
	}
	return b
}

// IfT 类似三目运算, 为了避免a, b被运算的问题，a, b可以输入函数，这样只有在需要的时候才会运行
func IfT(e bool, a, b any) any {
	if e {
		if reflect.TypeOf(a).Kind() == reflect.Func {
			return a.(func() any)()
		}
		return a
	}
	if reflect.TypeOf(b).Kind() == reflect.Func {
		return b.(func() any)()
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

// IsNil 指针是否为nil，非指针类型返回false
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	vOf := reflect.ValueOf(v)
	return vOf.Kind() == reflect.Ptr && vOf.IsNil()
}

// IsZero 判断是否为零值
func IsZero(v any) bool {
	// 指针为nil
	if IsNil(v) {
		return true
	}

	// 非指针类型，深度比较
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}

// IsZeroT 判断是否为零值，T必须是可比较的类型
func IsZeroT[T comparable](v T) bool {
	var zero T
	return v == zero
}

// NestAccess 递归访问map/struct/slice
// keys 是递归的key
// 比如: NestAccess({"a": {"b": [{"c": "string"}]}}, "a", "b", "0", "c")  ==> string
func NestAccess(from any, keys ...string) (any, error) {
	valueOf := reflect.ValueOf(from)
	for _i, k := range keys {
		if valueOf.IsNil() {
			return nil, errors.Errorf("have no key at %v", keys[:_i])
		} else if valueOf.Kind() == reflect.Ptr {
			valueOf = valueOf.Elem()
		}
		switch valueOf.Kind() {
		case reflect.Slice:
			i := conv.Atoi(k, -1)
			if i < 0 || i >= valueOf.Len() {
				return nil, errors.Errorf("have no key at %v", keys[:_i])
			} else {
				valueOf = valueOf.Index(i).Elem()
			}
		case reflect.Map:
			valueOf = valueOf.MapIndex(reflect.ValueOf(k)).Elem()
		case reflect.Struct:
			valueOf = valueOf.FieldByName(k).Elem()
		default:
			return nil, errors.Errorf("no key at %v", keys[:_i])
		}
	}

	return valueOf.Interface(), nil
}

func CopyFrom[T any](src []T) []T {
	var dest []T = make([]T, len(src))
	copy(dest, src)
	return dest
}
