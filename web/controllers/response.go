package controllers

import "fmt"

type Result struct {
	Code     interface{}         `json:"code"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Duration float64     `json:"duration"`
	At       int64       `json:"at"`
}

type IResponseException interface {
	error
	GetCode() interface{}
	GetStatusCode() int
	GetMessage() string
}


type ResponseException struct {
	Code    interface{}
	StatusCode int
	Message string
}

func NewResponseException(code interface{}, statusCode int, message string) *ResponseException {
	return &ResponseException{
		Code:    code,
		StatusCode:  statusCode,
		Message: message,
	}
}

func (e *ResponseException) GetCode() interface{} {
	return e.Code
}

func (e *ResponseException) Error() string {
	return fmt.Sprintf("[%v]: %s", e.Code, e.Message)
}

func (e *ResponseException) GetStatusCode() int {
	return e.StatusCode
}

func (e *ResponseException) GetMessage() string {
	return e.Message
}



