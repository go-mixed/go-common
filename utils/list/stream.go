package listUtils

import (
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
	"strings"
)

// 优先使用golang.org/x/exp/slices包中的方法
// slices.Index 返回第一个等于findMe的索引，找不到返回-1
// slices.IndexFunc 返回第一个满足fn的索引，找不到返回-1
// slices.Clip 返回一个新的slice，包含原slice中从start到end的元素
// slices.Clone 复制slice
// slices.BinarySearch 二分查找，返回第一个等于findMe的索引，找不到返回-1
// slices.BinarySearchFunc 二分查找，返回第一个满足fn的索引，找不到返回-1
// slices.Compact
// slices.CompactFunc
// slices.Compare 比较两个slice，返回-1，0，1
// slices.Equal 判断两个slice是否相等
// slices.EqualFunc 判断两个slice是否相等，使用自定义的比较函数
// slices.Contains 判断slice中是否包含findMe
// slices.ContainsFunc 判断slice中是否包含满足fn的元素
// slices.Delete 删除slice中满足条件的元素
// slices.Grow 将slice扩容到指定大小
// slices.Insert 在slice的指定位置插入元素
// slices.IsSorted 判断slice是否已经排序
// slices.IsSortedFunc 判断slice是否已经排序，使用自定义的比较函数
// slices.Sort 对slice进行排序
// slices.SortFunc 对slice进行排序，使用自定义的比较函数
// slices.SortStableFunc 对slice进行排序，使用自定义的比较函数
// slices.Replace 将slice中所有等于old的元素替换为new

// LastIndex 返回最后一个等于findMe的索引，找不到返回-1
func LastIndex[V comparable](slice []V, findMe V) int {
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
func Remove[V comparable](slice []V, removeMe V) []V {
	return Filter(slice, func(key int, value V) bool {
		return value != removeMe
	})
}

// Filter 通过fn筛选slice中的元素，fn返回true则保留
func Filter[V any](slice []V, fn func(key int, value V) bool) []V {
	var result []V
	for i := 0; i < len(slice); i++ {
		if fn(i, slice[i]) {
			result = append(result, slice[i])
		}
	}
	return result
}

// Min 返回slice中最小的值
func Min[V constraints.Ordered](s ...V) V {
	if len(s) == 0 {
		var zero V
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
func Max[V constraints.Ordered](s ...V) V {
	if len(s) == 0 {
		var zero V
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
func Sum[V constraints.Ordered | constraints.Complex](s ...V) V {
	if len(s) == 0 {
		var zero V
		return zero
	}
	m := s[0]
	for _, v := range s {
		m += v
	}
	return m
}

// Avg 返回slice中所有元素的平均值。Complex类型不支持
func Avg[V constraints.Integer | constraints.Float](s ...V) V {
	if len(s) == 0 {
		var zero V
		return zero
	}
	return Sum(s...) / (V)(len(s))
}

// Unique 返回去重后的slice
func Unique[V comparable](s ...V) []V {
	var result []V
	for i, v := range s {
		// 如果v在result中的索引不是i（自己），说明v已经存在于result中
		if slices.Index(result, v) != i {
			result = append(result, v)
		}
	}
	return result
}

// Map 将slice中的元素通过fn映射为另一个slice
func Map[V any, R any](s []V, fn func(key int, value V) R) []R {
	var result []R
	for i, v := range s {
		result = append(result, fn(i, v))
	}
	return result
}

// Foreach 遍历slice中的元素，如果fn返回error，则停止遍历并返回error
func Foreach[V any](s []V, fn func(key int, value V) error) error {
	for i, v := range s {
		if err := fn(i, v); err != nil {
			return err
		}
	}
	return nil
}

// Reverse 返回一个反转的slice
func Reverse[V any](s []V) []V {
	var result []V
	for i := len(s) - 1; i >= 0; i-- {
		result = append(result, s[i])
	}
	return result
}

// Chunk 将slice按照size分块，返回一个二维slice
func Chunk[V any](s []V, size int) [][]V {
	var result [][]V
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		result = append(result, s[i:end])
	}
	return result
}
