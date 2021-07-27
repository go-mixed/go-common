package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

func BuildDefaultHttpClient(timeout time.Duration) *http.Client {

	transport := &http.Transport{
		MaxIdleConns: 10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	return &http.Client{
		Timeout:  timeout,
		Transport: transport,
	}

}
