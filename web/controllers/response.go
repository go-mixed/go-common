package controllers

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"io"
)

type IResponseException interface {
	error
	GetCode() int
	SetCode(int)
	GetStatusCode() int
	SetStatusCode(int)
	GetMessage() string
	SetMessage(string)
}

type ResponseException struct {
	Code       int
	StatusCode int
	Message    string
}

var _ IResponseException = (*ResponseException)(nil)

func NewResponseException(code int, statusCode int, message string) IResponseException {
	return &ResponseException{
		Code:       code,
		StatusCode: statusCode,
		Message:    message,
	}
}

func (e *ResponseException) GetCode() int {
	return e.Code
}

func (e *ResponseException) SetCode(code int) {
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
func ParseResult(j []byte, outData any) (*utils.Result, error) {
	result := &utils.Result{Code: 0}
	if err := textUtils.JsonUnmarshalFromBytes(j, result); err != nil {
		return nil, err
	}

	if !core.IsInterfaceNil(outData) && !core.IsInterfaceNil(result.Data) {
		d, _ := textUtils.JsonMarshalToBytes(result.Data)
		if err := textUtils.JsonUnmarshalFromBytes(d, outData); err != nil {
			return result, err
		}
	}

	// 如果返回的code不为0，那么返回错误
	if result.Code != 0 {
		return result, errors.Errorf("code: %v, message: %s", result.Code, result.Message)
	}

	return result, nil
}

// ParseResultFromReader 从reader中读取JSON内容并解析为 Result, 并且解析 Result.Data 为 outData
//
//	返回的 Result.Data是JSON原文，传入outData参数将对Result.Data进行解析
//	会关闭reader
func ParseResultFromReader(reader io.ReadCloser, outData any) (*utils.Result, error) {
	if reader == nil {
		return nil, errors.Errorf("reader is nil")
	}
	defer reader.Close()

	j, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return ParseResult(j, outData)
}
