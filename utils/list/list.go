package listUtils

import (
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
	"reflect"
	"strings"
)

// Find 类似slice.IndexOf, 需要传递fn来判断是否相等
// 找不到返回-1
func Find[T comparable](slice []T, fn func(value T) bool) int {
	for i := 0; i < len(slice); i++ {
		if fn(slice[i]) {
			return i
		}
	}
	return -1
}

// IndexOf 简化版slice.IndexOf
func IndexOf[T comparable](slice []T, findMe T) int {
	return Find(slice, func(value T) bool {
		return value == findMe
	})
}

// StrIndexOf 字符串数组的IndexOf
func StrIndexOf(slice []string, findMe string, ignoreCase bool) int {
	if ignoreCase {
		findMe = strings.ToLower(findMe)
	}
	return Find(slice, func(value string) bool {
		if ignoreCase {
			value = strings.ToLower(value)
		}
		return findMe == value
	})
}

func SliceRemove[T comparable](slice []T, removeMe T) []T {
	var result []T
	for i := 0; i < len(slice); i++ {
		if slice[i] != removeMe {
			result = append(result, slice[i])
		}
	}
	return result
}

// Min 返回slice中最小的值
func Min[T constraints.Ordered](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m > v {
			m = v
		}
	}
	return m
}

// Max 返回slice中最大的值
func Max[T constraints.Ordered](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m < v {
			m = v
		}
	}
	return m
}

// ToInterfaces 将any 转为 []any, 因为不能直接slice.([]any)
//
//	注意：在1.18之后允许转换
//	 此函数使用场景可以参照 SortDomains
func ToInterfaces(src any) []any {
	s := reflect.ValueOf(src)
	if s.Kind() == reflect.Ptr {
		s = s.Elem()
	}
	if s.Kind() != reflect.Slice {
		panic("ToInterfaces() given a non-slice type")
	}

	// Keep the distinction between nil and empty slice input
	if s.IsNil() {
		return nil
	}

	ret := make([]any, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

// InterfacesAs 将一个src(类型为[]any)的值写到dest指针中
func InterfacesAs(src []any, dest any) error {
	valueOf := reflect.ValueOf(dest)

	// 判断是否是Slice的指针
	if k := valueOf.Kind(); k != reflect.Ptr {
		return errors.Errorf("expected pointer, got %v", k)
	}
	// 判断是否是Slice
	if k := valueOf.Elem().Kind(); k != reflect.Slice {
		return errors.Errorf("expected pointer to slice, got %v", k)
	}

	// 新建Slice对象
	typeOf := valueOf.Elem().Type().Elem()
	newSlice := reflect.MakeSlice(reflect.SliceOf(typeOf), len(src), len(src))

	for i, s := range src {
		newSlice.Index(i).Set(reflect.ValueOf(s))
	}

	valueOf.Elem().Set(newSlice)

	return nil
}

// InterfaceAs 将一个src(类型为Slice)的值写到dest指针中
// var a = []string{"a", "b"}
// var b []string
// InterfaceAs(a, &b)
func InterfaceAs(src any, dest any) error {
	return InterfacesAs(ToInterfaces(src), dest)
}
