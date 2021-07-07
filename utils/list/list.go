package list

import (
	"fmt"
	"reflect"
	"strings"
)

// Find 类似slice.IndexOf, 需要传递fn来判断是否相等
// 注意 这是反射实现的函数, 会降低运行性能
func Find(slice interface{}, fn func(value interface{}) bool) int {
	s := reflect.ValueOf(slice)
	if s.Kind() == reflect.Slice {
		for index := 0; index < s.Len(); index++ {
			if fn(s.Index(index).Interface()) {
				return index
			}
		}
	}
	return -1
}

// IndexOf 简化版slice.IndexOf，需要完全相等才会返回index
// 注意: 使用反射会降低运行性能, 尽量使用: StrIndexOf, IntIndexOf
func IndexOf(slice interface{}, findMe interface{}) int {
	return Find(slice, func(value interface{}) bool {
		return value == findMe
	})
}
// StrIndexOf 字符串数组的IndexOf
func StrIndexOf(slice []string, findMe string, ignoreCase bool) int {
	if ignoreCase {
		findMe = strings.ToLower(findMe)
	}
	for i := 0; i < len(slice); i++ {
		v := slice[i]
		if ignoreCase {
			v = strings.ToLower(v)
		}
		if findMe == v {
			return i
		}
	}

	return -1
}

// IntIndexOf int型数组的IndexOf
func IntIndexOf(slice []int, findMe int) int {
	for i := 0; i < len(slice); i++ {
		v := slice[i]

		if findMe == v {
			return i
		}
	}

	return -1
}

// ToInterfaces 将interface{} 转为 []interface{}, 因为不能直接slice.([]interface{}) 此函数使用场景可以参照 SortDomains
func ToInterfaces(src interface{}) []interface{} {
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

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

// InterfacesAs 将一个src(类型为[]interface{})的值写到dest指针中
func InterfacesAs(src []interface{}, dest interface{}) error {
	valueOf := reflect.ValueOf(dest)

	// 判断是否是Slice的指针
	if k := valueOf.Kind(); k != reflect.Ptr {
		return fmt.Errorf("expected pointer, got %v", k)
	}
	// 判断是否是Slice
	if k := valueOf.Elem().Kind(); k != reflect.Slice {
		return fmt.Errorf("expected pointer to slice, got %v", k)
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
func InterfaceAs(src interface{}, dest interface{}) error {
	return InterfacesAs(ToInterfaces(src), dest)
}
