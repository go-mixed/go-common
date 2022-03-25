package utils

import (
	"fmt"
	"go-common/utils/conv"
	"reflect"
)

// MapKeys 获取map的所有keys MapKeys({1: 'a', 2: 'c'})
func MapKeys[K comparable, V any](data map[K]V) []K {
	keys := make([]K, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys
}

// MapValues 获取map的所有values MapValues({1: 'a', 2: 'c'})
func MapValues[K comparable, V any](data map[K]V) []V {
	values := make([]V, 0, len(data))
	for _, v := range data {
		values = append(values, v)
	}
	return values
}

func ToMap(data any, tag string) (map[string]any, error) {
	var result = map[string]any{}

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

type KV struct {
	Key   string
	Value []byte
}

type KVs []*KV

func (s KVs) Append(k string, v []byte) KVs {
	return append(s, &KV{
		Key:   k,
		Value: v,
	})
}

func (s KVs) Add(_new KVs) KVs {
	return append(s, _new...)
}

func (s KVs) Keys() []string {
	var keys []string
	for _, kv := range s {
		keys = append(keys, kv.Key)
	}
	return keys
}

func (s KVs) Values() [][]byte {
	var values [][]byte
	for _, kv := range s {
		values = append(values, kv.Value)
	}
	return values
}

func NewKV(key string, value []byte) *KV {
	return &KV{Key: key, Value: value}
}
