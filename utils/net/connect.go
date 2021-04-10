package net

import (
	"net"
	"time"
)

func TryTcpConnect(addr string, timeout time.Duration) (ok bool, duration time.Duration, err error) {
	return TryConnect("tcp", addr, timeout)
}

func TryUdpConnect(addr string, timeout time.Duration) (ok bool, duration time.Duration, err error) {
	return TryConnect("udp", addr, timeout)
}

func TryConnect(protocol string, addr string, timeout time.Duration) (ok bool, duration time.Duration, err error) {
	n := time.Now()
	c, err := net.DialTimeout(protocol, addr, timeout)
	if err != nil {
		ok = false
		return
	}

	defer c.Close()
	ok = true
	duration = time.Now().Sub(n)
	return
}
