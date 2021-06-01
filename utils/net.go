package utils

import (
	"net"
)

// IsValidIP parses input string for ip address validity.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}
