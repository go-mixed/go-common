package controllers

import (
	"fmt"
	"go-common/utils/core"
	text_utils "go-common/utils/text"
	"io"
	"io/ioutil"
)

type Result struct {
	Code     interface{} `json:"code"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Duration float64     `json:"duration"`
	At       int64       `json:"at"`
}

type IResponseException interface {
	error
	GetCode() interface{}
	SetCode(interface{})
	GetStatusCode() int
	SetStatusCode(int)
	GetMessage() string
	SetMessage(string)
}

type ResponseException struct {
	Code       interface{}
	StatusCode int
	Message    string
}

func NewResponseException(code interface{}, statusCode int, message string) *ResponseException {
	return &ResponseException{
		Code:       code,
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *ResponseException) GetCode() interface{} {
	return e.Code
}

func (e *ResponseException) SetCode(code interface{}) {
	e.Code = code
}

func (e *ResponseException) Error() string {
	return fmt.Sprintf("[%v]: %s", e.Code, e.Message)
}

func (e *ResponseException) GetStatusCode() int {
	return e.StatusCode
}

func (e *ResponseException) SetStatusCode(statusCode int) {
	e.StatusCode = statusCode
}

func (e *ResponseException) GetMessage() string {
	return e.Message
}

func (e *ResponseException) SetMessage(message string) {
	e.Message = message
}

// ParseResult 读取JSON内容解析为 Result, 并且解析 Result.Data 为 outData
func ParseResult(j []byte, outData interface{}) (*Result, error) {
	result := &Result{}
	if err := text_utils.JsonUnmarshalFromBytes(j, result); err != nil {
		return nil, err
	}

	if !core_utils.IsInterfaceNil(outData) && !core_utils.IsInterfaceNil(result.Data) {
		d, _ := text_utils.JsonMarshalToBytes(result.Data)
		if err := text_utils.JsonUnmarshalFromBytes(d, outData); err != nil {
			return result, err
		}
	}

	return result, nil
}

// ParseResultFromReader 从reader中读取JSON内容并解析为 Result, 并且解析 Result.Data 为 outData
// 会关闭reader
func ParseResultFromReader(reader io.ReadCloser, outData interface{}) (*Result, error) {
	defer reader.Close()

	j, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return ParseResult(j, outData)
}
