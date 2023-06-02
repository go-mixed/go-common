package httpUtils

import (
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/conv"
	"gopkg.in/go-mixed/go-common.v1/utils/list"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"math"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultTimestampUnit = time.Second      // 默认时间戳单位，支持：time.Second, time.Millisecond, time.Microsecond, time.Nanosecond
const defaultSignedFields = "timestamp,nonce" // 默认签名字段

// BaseSignature 签名基础结构，继承此结构体，可实现签名功能
//
//		type A struct {
//			BaseSignature
//			B string `json:"b"`
//			C string `json:"c"`
//		}
//		a := &A{}
//	 a.SetSignedFields("b", "c")
//		a.BuildSignatures(a, "SecretKey", false)
//		a.CheckSignatures(a, "SecretKey", false)
type BaseSignature struct {
	Timestamp     int64 `json:"timestamp" form:"timestamp" query:"timestamp"`
	timestampUnit time.Duration
	Nonce         string `json:"nonce" form:"nonce" query:"nonce"`
	Sign          string `json:"sign" form:"sign" query:"sign"`
	SignedFields  string `json:"signed_fields,omitempty" form:"signed_fields,omitempty" query:"signed_fields,omitempty"`
}

type iSignature interface {
	SetSign(sign string)
	GetSign() string
	SetTimestamp(now time.Time)
	GetTimestamp() time.Time
	SetTimestampUnit(unit time.Duration)
	SetNonce(nonce string)
	GetNonce() string
	SetSignedFields(fields ...string)
	GetSignedFields() []string
}

var _ iSignature = (*BaseSignature)(nil)

// SetTimestamp 设置时间戳
func (s *BaseSignature) SetTimestamp(now time.Time) {
	if !now.IsZero() {
		now = time.Now()
	}

	var timestampUnit = s.timestampUnit

switch1:
	switch timestampUnit {
	case time.Millisecond:
		s.Timestamp = now.UnixMilli()
	case time.Microsecond:
		s.Timestamp = now.UnixMicro()
	case time.Nanosecond:
		s.Timestamp = now.UnixNano()
	case time.Second:
		s.Timestamp = now.Unix()
	default:
		timestampUnit = defaultTimestampUnit
		goto switch1
	}
}

// GetTimestamp 获取时间戳
func (s *BaseSignature) GetTimestamp() time.Time {
	if s.Timestamp <= 0 {
		return time.Time{}
	}

	var timestampUnit = s.timestampUnit

switch1:
	switch timestampUnit {
	case time.Millisecond:
		return time.UnixMilli(s.Timestamp)
	case time.Microsecond:
		return time.UnixMicro(s.Timestamp)
	case time.Nanosecond:
		return time.Unix(s.Timestamp/1e9, s.Timestamp%1e9)
	case time.Second:
		return time.Unix(s.Timestamp, 0)
	default:
		timestampUnit = defaultTimestampUnit
		goto switch1
	}
}

// SetTimestampUnit 设置时间戳的单位
func (s *BaseSignature) SetTimestampUnit(unit time.Duration) {
	s.timestampUnit = unit
}

// SetNonce 设置噪音值
func (s *BaseSignature) SetNonce(nonce string) {
	s.Nonce = strings.TrimSpace(nonce)
}

// GetNonce 获取噪音值
func (s *BaseSignature) GetNonce() string {
	return strings.TrimSpace(s.Nonce)
}

// SetSign 设置签名的值
func (s *BaseSignature) SetSign(sign string) {
	s.Sign = strings.TrimSpace(sign)
}

// GetSign 获取签名的值
func (s *BaseSignature) GetSign() string {
	return strings.TrimSpace(s.Sign)
}

// SetSignedFields 设置需要签名的字段，注意：字段名需要是Struct中字段tag名，即json:"xxx"中的xxx
func (s *BaseSignature) SetSignedFields(fields ...string) {
	if len(fields) > 0 {
		// 添加签名的基础字段，并排序
		fields = append(fields, strings.Split(defaultSignedFields, ",")...)
		sort.Strings(fields)
		fields = listUtils.Unique(fields...)
		s.SignedFields = strings.Join(fields, ",")
	} else {
		s.SignedFields = ""
	}
}

// GetSignedFields 获取需要签名的字段，注意：字段名是Struct中字段tag名，即json:"xxx"中的xxx
func (s *BaseSignature) GetSignedFields() []string {
	return strings.Split(strings.TrimSpace(s.SignedFields), ",")
}

// BuildSignature 生成签名
//
// obj: 需要签名的对象，BaseSignature的子类
// secretKey: 签名密钥
// withBlank: 是否包含空值字段
func (s *BaseSignature) BuildSignature(obj iSignature, secretKey string, withBlank bool) {
	// 设置基本字段
	obj.SetTimestamp(time.Now())
	obj.SetNonce(conv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(999999-100000) + 100000)) // >=6位数字

	// 将对象转换为url.Values
	values := ToUrlValues(obj, "json")
	delete(values, "sign")

	obj.SetSign(CalcSignature(secretKey, values, obj.GetSignedFields(), withBlank))
}

// CheckSignature 验证签名。返回：是否验证通过，错误信息
//
// obj: 需要签名的对象，BaseSignature的子类
// secretKey: 签名密钥
// withBlank: 是否包含空值字段
func (s *BaseSignature) CheckSignature(obj iSignature, secretKey string, withBlank bool) (bool, error) {
	if obj.GetSign() == "" {
		return false, errors.Errorf("sign is empty. query: %v", obj)
	}

	nonce := obj.GetNonce()
	timestamp := obj.GetTimestamp()

	if len(nonce) < 6 {
		return false, errors.Errorf("nonce length is too short, must >= 6. query: %+v", obj)
	}

	if math.Abs(time.Since(timestamp).Seconds()) >= 60. {
		return false, errors.Errorf("signature is expired: %s. query: %+v", timestamp, obj)
	}

	signature := obj.GetSign()
	values := ToUrlValues(obj, "json")
	delete(values, "sign")

	if strings.EqualFold(CalcSignature(secretKey, values, obj.GetSignedFields(), withBlank), signature) {
		return true, nil
	}

	return false, errors.Errorf("the signature is invalid: %s", signature)
}

// CalcSignature 输入values（url.Values）的数据，计算为签名，如果withBlank为false，值为空白字符串的不参与签名
func CalcSignature(secretKey string, values url.Values, signedFields []string, withBlank bool) string {

	var buf strings.Builder

	// 获取有效的keys，并对keys排序
	keys := make([]string, 0, len(values))
	for key := range values {
		// 字段不在signedFields中，不参与签名；signedFields为空，所有字段都参与签名
		if len(signedFields) > 0 && listUtils.StrIndexOf(signedFields, key, true) < 0 {
			continue
		}
		// 删除空白
		if !withBlank && values.Get(key) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys) // key的正序排序

	//"k1=v1&k2=v2" + key
	for i, k := range keys {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(values.Get(k))
		if i != len(keys)-1 { // 末尾不用&
			buf.WriteString("&")
		}
	}
	buf.WriteString(secretKey)

	res := textUtils.Md5(buf.String())

	utils.GetGlobalILogger().Debugf("Sign %s ==> %s", buf.String(), res)
	return res
}

// BuildMapSignature 传入map和待签名的signedFields，生成签名并附加到map中
//
//	如果signedFields为空，则所有字段都参与签名
func BuildMapSignature(secretKey string, data map[string]any, signedFields []string) map[string]any {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(signedFields) > 0 {
		signedFields = append(signedFields, "ts", "nonce", "signed_fields")
		data["signed_fields"] = strings.Join(signedFields, ",")
	}

	data["ts"] = conv.I64toa(time.Now().Unix())
	data["nonce"] = conv.Itoa(r.Intn(999999-100000) + 100000)
	data["sign"] = CalcMapSignature(secretKey, data, signedFields, true)

	return data
}

// CalcMapSignature 输入map的数据，计算为签名，
//
//	如果字段不在signedFields中，不参与签名；signedFields为空，所有字段都参与签名
//	如果withBlank为false，值为空白字符串（所有类型会强制转换）的不参与签名
func CalcMapSignature(secretKey string, data map[string]any, signedFields []string, withBlank bool) string {

	var keys []string
	for k := range data {
		// 字段不在signedFields中，不参与签名；signedFields为空，所有字段都参与签名
		if len(signedFields) > 0 && listUtils.StrIndexOf(signedFields, k, true) < 0 {
			continue
		}

		if !withBlank && (data[k] == nil || data[k] == "") { // 预先判断一次，空值不参与签名
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys) // key的正序排序

	var buf strings.Builder

	for i, k := range keys {

		buf.WriteString(k)
		buf.WriteString("=")

		d := data[k]
		r := textUtils.ToString(d, false)
		if !withBlank && r == "" { // 其它类型的数据强转后，如果空值不参与签名
			continue
		}
		buf.WriteString(r)

		if i != len(keys)-1 { // 末尾不用&
			buf.WriteString("&")
		}
	}
	buf.WriteString(secretKey)
	res := textUtils.Md5(buf.String())

	utils.GetGlobalILogger().Debugf("Sign %s ==> %s", buf.String(), res)
	return res
}

// CheckMapSignature 校验map的数据是否正确，并且和map['sign']中的签名是否正确
//
//	会尝试从map中查找signed_fields，如果没有，则所有字段全参与签名
func CheckMapSignature(secretKey string, m map[string]any) (bool, error) {
	nonce, _ := m["nonce"].(string)
	signature, _ := m["sign"].(string)
	_signFields, _ := m["signed_fields"].(string)
	signedFields := strings.Split(_signFields, ",")
	timestamp := m["ts"].(int64)
	delete(m, "sign")

	if len(nonce) < 6 {
		j, _ := textUtils.JsonMarshal(m)
		return false, errors.Errorf("nonce length is too short, must >= 6. json: %s", j)
	}

	if math.Abs(float64(timestamp-time.Now().Unix())) >= 60 {
		j, _ := textUtils.JsonMarshal(m)
		return false, errors.Errorf("out of signature time(60s), json: %s", j)
	}

	_s := CalcMapSignature(secretKey, m, signedFields, true)
	if strings.EqualFold(_s, signature) {
		return true, nil
	}

	return false, errors.Errorf("the signature is invalid: %s", signature)
}
