package utils

import (
	"fmt"
	"go-common/utils/conv"
	"reflect"
)

// MapKeys 获取map的所有keys MapKeys({1: 'a', 2: 'c'}).([]int)
// 相比直接foreach来取 要慢接近8倍
func MapKeys(data interface{}) interface{} {
	vOf := reflect.ValueOf(data)
	if vOf.Kind() == reflect.Ptr {
		vOf = vOf.Elem()
	}
	if vOf.IsNil() || vOf.Kind() != reflect.Map {
		return nil
	}

	list := reflect.MakeSlice(reflect.SliceOf(vOf.Type().Key()), 0, 0)
	list = reflect.Append(list, vOf.MapKeys()...)

	return list.Interface()
}

func MapStringKeys(data map[string]string) []string {
	var list = make([]string, len(data))
	i := 0
	for k := range data {
		list[i] = k
		i++
	}
	return list
}

// MapValues 获取map的所有values MapValues({1: 'a', 2: 'c'}).([]string)
// 相比直接foreach来取 要慢接近8倍
func MapValues(data interface{}) interface{} {
	vOf := reflect.ValueOf(data)
	if vOf.Kind() == reflect.Ptr {
		vOf = vOf.Elem()
	}
	if vOf.IsNil() || vOf.Kind() != reflect.Map {
		return nil
	}

	list := reflect.MakeSlice(reflect.SliceOf(vOf.Type().Elem()), 0, 0)
	it := vOf.MapRange()
	for {
		if !it.Next() {
			break
		}
		list = reflect.Append(list, it.Value())
	}

	return list.Interface()
}

func MapStringValues(data map[string]string) []string {
	var list = make([]string, len(data))
	i := 0
	for _, v := range data {
		list[i] = v
		i++
	}
	return list
}

func ToMap(data interface{}, tag string) (map[string]interface{}, error) {
	var result = map[string]interface{}{}

	vOf := reflect.ValueOf(data)
	if data == nil || (vOf.Kind() == reflect.Ptr && vOf.IsNil()) {
		return result, nil
	}

	if vOf.Kind() == reflect.Ptr {
		vOf = vOf.Elem()
	}

	switch vOf.Kind() {
	case reflect.Map:
		for _, kOf := range vOf.MapKeys() {
			k := fmt.Sprintf("%v", kOf.Interface())
			result[k] = vOf.MapIndex(kOf).Interface()
		}
	case reflect.Struct:
		tOf := vOf.Type()
		for i := 0; i < tOf.NumField(); i++ {
			field := tOf.Field(i)
			name := field.Name
			if tag != "" {
				tagName, ok := field.Tag.Lookup(tag)
				if ok && tagName != "-" && tagName != "_" {
					name = tagName
				}
			}
			result[name] = vOf.Field(i).Interface()
		}
	case reflect.Slice:
		for i := 0; i < vOf.Len(); i++ {
			result[conv.Itoa(i)] = vOf.Index(i).Interface()
		}
	}

	return result, nil
}
