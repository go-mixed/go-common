package text_utils

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go-common/utils/core"
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
	return ListDecode(JsonUnmarshalFromBytes, jsonList, to)
}

func JsonUnmarshal(_json string, to any) error {
	return errors.WithStack(jsoniter.ConfigCompatibleWithStandardLibrary.UnmarshalFromString(_json, to))
}

func JsonMarshal(from any) (string, error) {
	j, err := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(from)
	return j, errors.WithStack(err)
}

func JsonUnmarshalFromBytes(_json []byte, to any) error {
	return errors.WithStack(jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(_json, to))
}

func JsonMarshalToBytes(from any) ([]byte, error) {
	j, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(from)
	return j, errors.WithStack(err)
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
