package mapUtils

// 优先使用golang.org/x/exp/maps中的方法
// maps.Keys 返回map中所有的key
// maps.Values 返回map中所有的value
// maps.DeleteFunc 删除map中满足条件的key
// maps.Equal 判断两个map是否相等
// maps.EqualFunc 判断两个map是否相等，使用自定义的比较函数
// maps.Clear 清空map
// maps.Clone 复制map
// maps.Copy 复制map

// Filter 通过fn筛选map中的元素，fn返回true则保留
func Filter[KT comparable, T any](m map[KT]T, fn func(key KT, value T) bool) map[KT]T {
	var result = make(map[KT]T)
	for k, v := range m {
		if fn(k, v) {
			result[k] = v
		}
	}
	return result
}

// Map 遍历map，对每个元素执行fn，返回一个新的map
func Map[T ~map[K]V, K comparable, V any, R any](data T, fn func(key K, value V) (K, R)) map[K]R {
	var result map[K]R
	for k, v := range data {
		nK, nV := fn(k, v)
		result[nK] = nV
	}
	return result
}

// Combine 将两个slice合并成一个map，第一个slice为key，第二个slice为value
func Combine[K comparable, V any](keys []K, values []V) map[K]V {
	if len(keys) != len(values) {
		panic("keys and values must have same length")
	}
	result := make(map[K]V)
	for i, k := range keys {
		result[k] = values[i]
	}
	return result
}

// Merge 将多个map合并成一个map，如果key重复，后面的map会覆盖前面的map
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
