//go:build !linux

package core

func IsInWSL() bool {
	return false
}

func IsInWSL2() bool {
	return false
}
