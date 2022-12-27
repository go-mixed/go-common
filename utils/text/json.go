package text_utils

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go-common/utils/core"
	"reflect"
	"strings"
)

// JsonListUnmarshal 将json字符串数组 转换成一个 []any
// 例子
// type User struct { Name string Age int}
// var users []User
// JsonListIntoSlicePtr([]string{"{\"Name\": \"a\", \"Age\": 20}", "{\"Name\": \"b\", \"Age\": 21}"}, &users)
func JsonListUnmarshal(jsonList []string, to any) error {
	var list [][]byte
	for _, _j := range jsonList {
		if _j == "" {
			list = append(list, nil)
		} else {
			list = append(list, []byte(_j))
		}
	}

	return JsonListUnmarshalFromBytes(list, to)
}

func JsonListUnmarshalFromBytes(jsonList [][]byte, to any) error {
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

	for _, _json := range jsonList {
		if _json == nil {
			newSlice = reflect.Append(newSlice, reflect.Zero(typeOfV))
			continue
		}

		newInstance := reflect.New(typeOfV).Elem()
		// 传递newInstance的指针给 json.Unmarshal
		if err := JsonUnmarshalFromBytes(_json, newInstance.Addr().Interface()); err != nil {
			return err
		}

		newSlice = reflect.Append(newSlice, newInstance)
	}

	toValue.Set(newSlice)
	return nil
}

func JsonUnmarshal(_json string, to any) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.UnmarshalFromString(_json, to)
}

func JsonMarshal(from any) (string, error) {
	return jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(from)
}

func JsonUnmarshalFromBytes(_json []byte, to any) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(_json, to)
}

func JsonMarshalToBytes(from any) ([]byte, error) {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(from)
}

// JsonExtractIntoPtr 将一个json转到to, 支持使用.递归访问json内的值进行转换
// to := struct{Name string}{}  JsonExtractIntoPtr({"a": {"b": [{"Name": "string"}]}}, &to, "a.b.0")
func JsonExtractIntoPtr(_json []byte, to any, label string) error {
	if label == "" {
		return JsonUnmarshalFromBytes(_json, to)
	}

	var m map[string]any
	if err := JsonUnmarshalFromBytes(_json, &m); err != nil {
		return err
	}

	j, err := core.NestAccess(m, strings.Split(label, ".")...)
	if err != nil {
		return err
	}

	b, err := JsonMarshalToBytes(j)
	if err != nil {
		return err
	}
	return JsonUnmarshalFromBytes(b, to)
}
