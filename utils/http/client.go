package http_utils

import (
	"net/http"
	"time"
)

func DefaultHttpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: NewHttpTransport(DefaultTransportOptions()),
	}
}
