package text_utils

import (
	"bytes"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
)

func GbkToUtf8FromBytes(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func GbkToUtf8(s string) (string, error) {
	b, err := GbkToUtf8FromBytes([]byte(s))
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func Utf8ToGbkFromBytes(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func Utf8ToGbk(s string) (string, error) {
	b, err := Utf8ToGbkFromBytes([]byte(s))
	if err != nil {
		return "", err
	}

	return string(b), nil
}
