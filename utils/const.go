package utils

// Result common result of response
type Result struct {
	Code     any     `json:"code"`
	Message  string  `json:"message,omitempty"`
	Data     any     `json:"data,omitempty"`
	Duration float64 `json:"duration"`
	At       int64   `json:"at"`
}

type SResult struct {
	Code     int     `json:"code"`
	Message  string  `json:"message,omitempty"`
	Data     any     `json:"data,omitempty"`
	Duration float64 `json:"duration"`
	At       int64   `json:"at"`
}
