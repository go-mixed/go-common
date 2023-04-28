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

type SignBaseFields struct {
	TimeStamp    int64  `json:"ts" form:"ts" query:"ts"`
	Nonce        string `json:"nonce" form:"nonce" query:"nonce"`
	Sign         string `json:"sign" form:"sign" query:"sign"`
	SignedFields string `json:"signed_fields,omitempty" form:"signed_fields,omitempty" query:"signed_fields,omitempty"`
}

// MakeSignature 传入values（url.Values）, 生成签名并附加到 values 中
//
//	如果 signedFields 为空，则所有字段都参与签名
func MakeSignature(secretKey string, values url.Values, signedFields []string) url.Values {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(signedFields) > 0 {
		signedFields = append(signedFields, "ts", "nonce", "signed_fields")
		values.Set("signed_fields", strings.Join(signedFields, ","))
	}

	values.Set("ts", conv.I64toa(time.Now().Unix()))
	values.Set("nonce", conv.Itoa(r.Intn(999999-100000)+100000)) // >=6位数字
	values.Set("sign", CalcSignature(secretKey, values, signedFields, true))

	return values
}

// MakeMapSignature 传入map和待签名的signedFields，生成签名并附加到map中
//
//	如果signedFields为空，则所有字段都参与签名
func MakeMapSignature(secretKey string, data map[string]any, signedFields []string) map[string]any {
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

// MakeArgumentsSignature 传入kv对，生成签名并返回 SignBaseFields
//
//	kvs的长度必须为偶数，key => value，并且所有的key都会参与签名
func MakeArgumentsSignature(secretKey string, kvs ...any) *SignBaseFields {
	return (&SignBaseFields{}).BuildSignature(secretKey, kvs...)
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

// CalcSignature 输入values（url.Values）的数据，计算为签名，如果withBlank为false，值为空白字符串的不参与签名
func CalcSignature(secretKey string, values url.Values, signedFields []string, withBlank bool) string {

	var buf strings.Builder

	// 对keys排序
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

// CheckSignature 校验values的数据是否正确，并且和values['sign']中的签名是否正确
//
//	会尝试从values中查找signed_fields，如果没有，则所有字段全参与签名
func CheckSignature(secretKey string, values url.Values) (bool, error) {
	nonce := strings.TrimSpace(values.Get("nonce"))
	signature := strings.TrimSpace(values.Get("sign"))
	timestamp := conv.Atoi64(values.Get("ts"), 0)
	signedFields := strings.Split(values.Get("signed_fields"), ",")
	values.Del("sign")

	if len(nonce) < 6 {
		return false, errors.Errorf("nonce length is too short, must >= 6. query: %s", values.Encode())
	}

	if math.Abs(float64(timestamp-time.Now().Unix())) >= 60 {
		return false, errors.Errorf("out of time: %d. query: %s", timestamp, values.Encode())
	}

	_s := CalcSignature(secretKey, values, signedFields, true)
	if strings.EqualFold(_s, signature) {
		return true, nil
	}

	return false, errors.Errorf("the signature is invalid: %s", signature)
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

func argsToSignatureValues(kvs ...any) (url.Values, []string) {
	if len(kvs)%2 != 0 {
		panic("Arguments Signature: kvs must be even, key => value")
	}

	signedFields := []string{"ts", "nonce", "signed_fields"}

	var values url.Values = url.Values{}
	for i := 0; i < len(kvs); i += 2 {
		k, _ := kvs[i].(string)
		values.Set(k, textUtils.ToString(kvs[i+1], false))
		signedFields = append(signedFields, k)
	}

	return values, signedFields
}

func (f *SignBaseFields) validate() error {
	if len(f.Nonce) < 6 {
		return errors.Errorf("nonce length is too short, must >= 6")
	}

	if math.Abs(float64(f.TimeStamp-time.Now().Unix())) >= 60 {
		return errors.Errorf("out of signature time(60s)")
	}

	return nil
}

func (f *SignBaseFields) BuildSignature(secretKey string, kvs ...any) *SignBaseFields {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	var fields *SignBaseFields = f
	if f == nil {
		fields = &SignBaseFields{}
	}
	values, signedFields := argsToSignatureValues(kvs...)

	fields.TimeStamp = time.Now().Unix()
	fields.Nonce = conv.Itoa(r.Intn(999999-100000) + 100000)
	fields.SignedFields = strings.Join(signedFields, ",")
	values.Set("ts", conv.I64toa(fields.TimeStamp))
	values.Set("nonce", fields.Nonce)
	values.Set("signed_fields", fields.SignedFields)

	fields.Sign = CalcSignature(secretKey, values, signedFields, true)
	return fields
}

func (f *SignBaseFields) CheckSignature(secretKey string, kvs ...any) error {
	if f == nil {
		return errors.Errorf("SignBaseFields is nil, did you set the sign, ts, nonce, [signed_fields] fields?")
	}

	if err := f.validate(); err != nil {
		return err
	}

	values, signedFields := argsToSignatureValues(kvs...)
	values.Set("ts", conv.I64toa(f.TimeStamp))
	values.Set("nonce", f.Nonce)
	values.Set("signed_fields", f.SignedFields)
	values.Del("sign")

	_s := CalcSignature(secretKey, values, signedFields, true)
	if strings.EqualFold(_s, f.Sign) {
		return nil
	}

	return errors.Errorf("the signature is invalid: %s", f.Sign)
}
