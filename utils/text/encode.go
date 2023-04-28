package textUtils

import (
	"github.com/pkg/errors"
	"reflect"
)

type EncoderFunc func(any) ([]byte, error)
type DecoderFunc func([]byte, any) error

// ListDecodeAny 将多条数据list解码为数组to，to需要为slice类型，decodeFunc可以传入gob、json、yaml的decode函数
func ListDecodeAny(decodeFunc DecoderFunc, list [][]byte, to any) error {
	toValue := reflect.ValueOf(to)
	if toValue.Kind() == reflect.Ptr {
		toValue = toValue.Elem()
	} else {
		return errors.Errorf("parameter \"to\" must be a ptr")
	}

	if toValue.Kind() != reflect.Slice {
		return errors.Errorf("parameter \"to\" must be a slice ptr")
	}

	// []any 得到any的类型
	typeOfV := toValue.Type().Elem()

	newSlice := reflect.MakeSlice(reflect.SliceOf(typeOfV), 0, 0)

	for _, _json := range list {
		if _json == nil {
			newSlice = reflect.Append(newSlice, reflect.Zero(typeOfV))
			continue
		}

		newInstance := reflect.New(typeOfV).Elem()
		// 传递newInstance的指针给 json.Unmarshal
		if err := decodeFunc(_json, newInstance.Addr().Interface()); err != nil {
			return err
		}

		newSlice = reflect.Append(newSlice, newInstance)
	}

	toValue.Set(newSlice)
	return nil
}

// ListDecode 使用泛型的方法依次将list的子项转为to的子项，比上面效率高
func ListDecode[T any](decodeFunc DecoderFunc, list [][]byte, to *[]T) error {
	var _to []T
	var err error
	for _, buf := range list {
		var t T
		if err = decodeFunc(buf, &t); err != nil {
			return err
		}
		_to = append(_to, t)
	}
	*to = _to
	return nil
}
