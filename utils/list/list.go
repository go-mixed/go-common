package list

import "reflect"

func Find(slice interface{}, f func(value interface{}) bool) int {
	s := reflect.ValueOf(slice)
	if s.Kind() == reflect.Slice {
		for index := 0; index < s.Len(); index++ {
			if f(s.Index(index).Interface()) {
				return index
			}
		}
	}
	return -1
}

func IndexOf(slice interface{}, findMe interface{}) int {
	return Find(slice, func(value interface{}) bool {
		return value == findMe
	})
}
