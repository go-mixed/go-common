package mapUtils

import (
	"fmt"
	"gopkg.in/go-mixed/go-common.v1/utils/conv"
	"reflect"
	"strings"
)

// ToMap 将map/struct/slice转换为map[string]any
//
//	如果是map，则尝试转换为map[string]any
//	如果是struct，则key是字段的tag。注意：私有字段不会被转换；匿名字段会被展开；子集不会展开
func ToMap(data any, tag string) (map[string]any, error) {
	var result = map[string]any{}

	if data == nil {
		return result, nil
	}

	vOf := reflect.ValueOf(data)
	if vOf.Kind() == reflect.Ptr && vOf.IsNil() {
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
			// 如果是私有字段，跳过
			if !field.IsExported() {
				continue
			}
			name := field.Name
			if tag != "" {
				tagName, ok := field.Tag.Lookup(tag)
				if ok && tagName != "-" && tagName != "_" {
					segments := strings.Split(tagName, ",")
					name = segments[0]
				}
			}
			// 如果是匿名字段，展开
			if field.Anonymous {
				children, err := ToMap(vOf.Field(i).Interface(), tag)
				if err != nil {
					return nil, err
				}
				for k := range children {
					result[k] = children[k]
				}
				continue
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
