package controllers

type Result struct {
	Code     int         `json:"code"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	Duration float64     `json:"duration"`
	At       int64       `json:"at"`
}

type ResponseException struct {
	Code    int
	Message string
}

func NewResponseException(code int, message string) *ResponseException {
	return &ResponseException{
		Code:    code,
		Message: message,
	}
}
