package httpUtils

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils/conv"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var HttpReason = map[int]string{

	100: "Continue",
	101: "Switching Protocols",
	102: "Processing", // RFC2518
	103: "Early Hints",
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	207: "Multi-Status",     // RFC4918
	208: "Already Reported", // RFC5842
	226: "IM Used",          // RFC3229
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	307: "Temporary Redirect",
	308: "Permanent Redirect", // RFC7238
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Timeout",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Payload Too Large",
	414: "URI Too Long",
	415: "Unsupported Media Type",
	416: "Range Not Satisfiable",
	417: "Expectation Failed",
	418: "I\"m a teapot",                   // RFC2324
	421: "Misdirected Request",             // RFC7540
	422: "Unprocessable Entity",            // RFC4918
	423: "Locked",                          // RFC4918
	424: "Failed Dependency",               // RFC4918
	425: "Too Early",                       // RFC-ietf-httpbis-replay-04
	426: "Upgrade Required",                // RFC2817
	428: "Precondition Required",           // RFC6585
	429: "Too Many Requests",               // RFC6585
	431: "Request Header Fields Too Large", // RFC6585
	451: "Unavailable For Legal Reasons",   // RFC7725
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Timeout",
	505: "HTTP Version Not Supported",
	506: "Variant Also Negotiates",         // RFC2295
	507: "Insufficient Storage",            // RFC4918
	508: "Loop Detected",                   // RFC5842
	510: "Not Extended",                    // RFC2774
	511: "Network Authentication Required", // RFC6585
}

// ValuesToJson 是一个将Query转化为Json的函数
func ValuesToJson(values *url.Values) []byte {
	var _values = map[string]any{}
	for key, val := range *values {
		_values[key] = core.IfT(len(val) <= 1, val[0], val)
	}

	buf, err := textUtils.JsonMarshalToBytes(_values)

	if err != nil {
		return nil
	}

	return buf
}

// ToUrlValues 将map/struct/slice/string转化为url.Values。
//
//	如果是string，则会尝试使用url.ParseQuery进行转换
//	如果是map，则会将key/value转换为string，tag传入""
//	如果是struct，则会将struct的字段的tag名（比如json）作为key。注意：匿名字段会展开；子Struct一律会转换为json
func ToUrlValues(data any, tag string) url.Values {
	var result url.Values = url.Values{}
	if data == nil {
		return result
	}

	vOf := reflect.ValueOf(data)
	if vOf.Kind() == reflect.Ptr && vOf.IsNil() {
		return result
	}

	if vOf.Kind() == reflect.Ptr {
		vOf = vOf.Elem()
	}

	switch vOf.Kind() {
	case reflect.String:
		result, _ = url.ParseQuery(vOf.String())
	case reflect.Map:
		for _, kOf := range vOf.MapKeys() {
			k := fmt.Sprintf("%v", kOf.Interface())
			result.Set(k, textUtils.ToString(vOf.MapIndex(kOf).Interface(), true))
		}
	case reflect.Struct:
		tOf := vOf.Type()
		for i := 0; i < tOf.NumField(); i++ {
			field := tOf.Field(i)
			// 如果是私有字段，则跳过
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
			// 如果是匿名字段，则展开
			if field.Anonymous {
				c := ToUrlValues(vOf.Field(i).Interface(), tag)
				for k := range c {
					result.Set(k, c.Get(k))
				}
				continue
			}

			result.Set(name, textUtils.ToString(vOf.Field(i).Interface(), true))
		}
	case reflect.Slice:
		for i := 0; i < vOf.Len(); i++ {
			result.Set(conv.Itoa(i), textUtils.ToString(vOf.Index(i).Interface(), true))
		}
	}

	return result
}

// MapToUrlValues 一个简单的map -> url.Values, 需要传入需要转换的字节名列表
// map的value尽量为字符串/数字/浮点数字, 不然转换出来的结果可能不符合预期
func MapToUrlValues(data map[string]any, includeFields []string) url.Values {
	values := url.Values{}
	if len(includeFields) <= 0 {
		return values
	}

	for _, k := range includeFields {
		v := data[k]
		values.Add(k, textUtils.ToString(v, true))
	}
	return values
}

// CloseResponseWriter 关闭ResponseWrite
func CloseResponseWriter(w http.ResponseWriter) error {
	hj, ok := w.(http.Hijacker)

	// The rw can't be hijacked, return early.
	if !ok {
		return errors.Errorf("can't hijack ResponseWriter")
	}

	// Hijack the rw.
	conn, _, err := hj.Hijack()
	if err != nil {
		return err
	}

	// Close the hijacked raw tcp connection.
	if err := conn.Close(); err != nil {
		return err
	}

	return nil
}

// Expects ascii encoded strings - from output of urlEncodePath
func percentEncodeSlash(s string) string {
	return strings.Replace(s, "/", "%2F", -1)
}

// QueryEncode - encodes query values in their URL encoded form. In
// addition to the percent encoding performed by urlEncodePath() used
// here, it also percent encodes '/' (forward slash)
func QueryEncode(v url.Values) string {
	if v == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		prefix := percentEncodeSlash(EncodePath(k)) + "="
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(prefix)
			buf.WriteString(percentEncodeSlash(EncodePath(v)))
		}
	}
	return buf.String()
}

// if object matches reserved string, no need to encode them
var reservedObjectNames = regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")

// EncodePath encode the strings from UTF-8 byte representations to HTML hex escape sequences
//
// This is necessary since regular url.Parse() and url.Encode() functions do not support UTF-8
// non english characters cannot be parsed due to the nature in which url.Encode() is written
//
// This function on the other hand is a direct replacement for url.Encode() technique to support
// pretty much every UTF-8 character.
func EncodePath(pathName string) string {
	if reservedObjectNames.MatchString(pathName) {
		return pathName
	}
	var encodedPathname strings.Builder
	for _, s := range pathName {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' { // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/': // §2.3 Unreserved characters (mark)
			encodedPathname.WriteRune(s)
			continue
		default:
			_len := utf8.RuneLen(s)
			if _len < 0 {
				// if utf8 cannot convert return the same string as is
				return pathName
			}
			u := make([]byte, _len)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				_hex := hex.EncodeToString([]byte{r})
				encodedPathname.WriteString("%" + strings.ToUpper(_hex))
			}
		}
	}
	return encodedPathname.String()
}

// UnescapeQueries Escape encodedQuery string into unescaped list of query params, returns error
// if any while unescaping the values.
func UnescapeQueries(encodedQuery string) (unescapedQueries []string, err error) {
	for _, query := range strings.Split(encodedQuery, "&") {
		var unescapedQuery string
		unescapedQuery, err = url.QueryUnescape(query)
		if err != nil {
			return nil, err
		}
		unescapedQueries = append(unescapedQueries, unescapedQuery)
	}
	return unescapedQueries, nil
}

func SetRequestKeyValue(r *http.Request, key, value any) *http.Request {
	c := context.WithValue(r.Context(), key, value)
	return r.WithContext(c)
}

func GetRequestValue(r *http.Request, key any) any {
	return r.Context().Value(key)
}

func DomainFromRequestHost(host string) string {
	var domain = host
	if strings.Contains(host, ":") {
		domain, _, _ = net.SplitHostPort(host)
	}
	return domain
}
