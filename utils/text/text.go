package text_utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"go-common/utils/core"
	"reflect"
	"strings"
)

// WildcardMatchSimple - finds whether the text matches/satisfies the pattern string.
// supports only '*' wildcard in the pattern.
// considers a file system path as a flat name space.
func WildcardMatchSimple(pattern, name string) bool {
	if pattern == "" {
		return name == pattern
	}
	if pattern == "*" {
		return true
	}
	// Does only wildcard '*' match.
	return deepWildcardMatchRune([]rune(name), []rune(pattern), true)
}

// WildcardMatch -  finds whether the text matches/satisfies the pattern string.
// supports  '*' and '?' wildcards in the pattern string.
// unlike path.Match(), considers a path as a flat name space while matching the pattern.
// The difference is illustrated in the example here https://play.golang.org/p/Ega9qgD4Qz .
func WildcardMatch(pattern, name string) (matched bool) {
	if pattern == "" {
		return name == pattern
	}
	if pattern == "*" {
		return true
	}
	// Does extended wildcard '*' and '?' match.
	return deepWildcardMatchRune([]rune(name), []rune(pattern), false)
}

func deepWildcardMatchRune(str, pattern []rune, simple bool) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		default:
			if len(str) == 0 || str[0] != pattern[0] {
				return false
			}
		case '?':
			if len(str) == 0 && !simple {
				return false
			}
		case '*':
			return deepWildcardMatchRune(str, pattern[1:], simple) || // 当前str[0]的字符是否匹配*之后(pattern[1])的字符
				(len(str) > 0 && deepWildcardMatchRune(str[1:], pattern, simple)) // 上面没匹配到, 则继续用*匹配下一个(str[1])字符
		}
		str = str[1:]
		pattern = pattern[1:]
	}
	return len(str) == 0 && len(pattern) == 0
}

func Md5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

type Stringer interface {
	String() string
}

// ToString 将任意类型转为字符串, 标量或标量的子类型可以直接转, 其它转为json的字符串
// 注意: type ABC string 这种类型会走到default分支, 为了减少反射带来的性能负担, 对已知可以强转的类型, 可以自行强转: string(abc)
// otherTypeAsJson: 无法识别的type转换为json 不然会返回空字符串
func ToString(v any, otherTypeAsJson bool) string {
	// 先用 type assert检查, 支持标量, 速度更快
	switch v.(type) {
	case []rune:
		return string(v.([]rune))
	case []byte:
		return string(v.([]byte))
	case string:
		return v.(string)
	case bool:
		return core.If(v.(bool), "true", "false")
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, complex64, complex128:
		return fmt.Sprintf("%v", v)
	case Stringer:
		return v.(Stringer).String()
	case error:
		return v.(error).Error()
	default:
		// 针对 type ABC string 这种需要使用typeof.kind检查
		switch reflect.TypeOf(v).Kind() {
		case reflect.Bool:
			return core.If(reflect.ValueOf(v).Bool(), "true", "false")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
			reflect.String:
			return fmt.Sprintf("%v", v)
		default: // 均不符合 则使用json来处理
			if otherTypeAsJson {
				j, _ := JsonMarshal(v)
				return j
			} else {
				return ""
			}
		}
	}
}

// SubUntil 从str截取字符, 直到untilStr结束 (不包含untilStr)
// 比如: SubUntil("abc/def", "/") -> "abc", 3
// 如果返回-1, 则表示没有找到 untilStr
func SubUntil(str string, untilStr string) (string, int) {
	i := strings.Index(str, untilStr)
	if i < 0 {
		return "", -1
	}

	return str[0:i], i
}
