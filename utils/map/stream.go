package mapUtils

func Filter[KT comparable, T any](m map[KT]T, fn func(key KT, value T) bool) map[KT]T {
	var result = make(map[KT]T)
	for k, v := range m {
		if fn(k, v) {
			result[k] = v
		}
	}
	return result
}

// Keys 获取map的所有keys MapKeys({1: 'a', 2: 'c'})
func Keys[K comparable, V any](data map[K]V) []K {
	keys := make([]K, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// Values 获取map的所有values MapValues({1: 'a', 2: 'c'})
func Values[K comparable, V any](data map[K]V) []V {
	values := make([]V, 0, len(data))
	for _, v := range data {
		values = append(values, v)
	}
	return values
}
