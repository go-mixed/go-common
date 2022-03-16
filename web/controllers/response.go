package controllers

import (
	"fmt"
	"go-common/utils/core"
	"go-common/utils/text"
	"io"
	"io/ioutil"
)

type Result struct {
	Code     any     `json:"code"`
	Message  string  `json:"message,omitempty"`
	Data     any     `json:"data,omitempty"`
	Duration float64 `json:"duration"`
	At       int64   `json:"at"`
}

type IResponseException interface {
	error
	GetCode() any
	SetCode(any)
	GetStatusCode() int
	SetStatusCode(int)
	GetMessage() string
	SetMessage(string)
}

type ResponseException struct {
	Code       any
	StatusCode int
	Message    string
}

func NewResponseException(code any, statusCode int, message string) IResponseException {
	return &ResponseException{
		Code:       code,
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *ResponseException) GetCode() any {
	return e.Code
}

func (e *ResponseException) SetCode(code any) {
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
func ParseResult(j []byte, outData any) (*Result, error) {
	result := &Result{}
	if err := text_utils.JsonUnmarshalFromBytes(j, result); err != nil {
		return nil, err
	}

	if !core.IsInterfaceNil(outData) && !core.IsInterfaceNil(result.Data) {
		d, _ := text_utils.JsonMarshalToBytes(result.Data)
		if err := text_utils.JsonUnmarshalFromBytes(d, outData); err != nil {
			return result, err
		}
	}

	return result, nil
}

// ParseResultFromReader 从reader中读取JSON内容并解析为 Result, 并且解析 Result.Data 为 outData
// 会关闭reader
func ParseResultFromReader(reader io.ReadCloser, outData any) (*Result, error) {
	defer reader.Close()

	j, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return ParseResult(j, outData)
}
