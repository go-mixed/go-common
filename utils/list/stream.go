package listUtils

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"strings"
)

// LastIndex 返回最后一个等于findMe的索引，找不到返回-1
func LastIndex[T comparable](slice []T, findMe T) int {
	for i := len(slice) - 1; i >= 0; i-- {
		if slice[i] == findMe {
			return i
		}
	}
	return -1
}

// StrIndexOf 字符串数组的IndexOf，可以选择是否忽略大小写
func StrIndexOf(slice []string, findMe string, ignoreCase bool) int {
	if ignoreCase {
		findMe = strings.ToLower(findMe)
	}
	return slices.IndexFunc(slice, func(value string) bool {
		if ignoreCase {
			value = strings.ToLower(value)
		}
		return findMe == value
	})
}

// Remove 从slice中移除removeMe，注意：所有等于removeMe的元素都会被移除
func Remove[T comparable](slice []T, removeMe T) []T {
	return Filter(slice, func(key int, value T) bool {
		return value != removeMe
	})
}

// Filter 通过fn筛选slice中的元素，fn返回true则保留
func Filter[T any](slice []T, fn func(key int, value T) bool) []T {
	var result []T
	for i := 0; i < len(slice); i++ {
		if fn(i, slice[i]) {
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

// Sum 返回slice中所有元素的和
func Sum[T constraints.Ordered | constraints.Complex](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		m += v
	}
	return m
}

// Avg 返回slice中所有元素的平均值。Complex类型不支持
func Avg[T constraints.Integer | constraints.Float](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	return Sum(s...) / (T)(len(s))
}

// Unique 返回去重后的slice
func Unique[T comparable](s ...T) []T {
	var result []T
	for i, v := range s {
		// 如果v在result中的索引不是i（自己），说明v已经存在于result中
		if slices.Index(result, v) != i {
			result = append(result, v)
		}
	}
	return result
}

// Pluck 从slice中提取出元素的某个属性，返回一个新的slice。fn为提取函数，返回值为新slice的元素
func Pluck[T ~[]V, V any, R any](slice T, fn func(key int, value V) R) []R {
	var result []R
	for i, v := range slice {
		result = append(result, fn(i, v))
	}
	return result
}
