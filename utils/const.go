package utils

import "github.com/pkg/errors"

// Result common result of response
type Result struct {
	Code     int     `json:"code"`
	Message  string  `json:"message,omitempty"`
	Data     any     `json:"data,omitempty"`
	Duration float64 `json:"duration"`
	At       int64   `json:"at"`
}

var ErrForEachBreak = errors.New("for each break")
var ErrForEachQuit = errors.New("for each quit")
