package core

import (
	"io/ioutil"
	"os"
	"strings"
)

// IsInWSL 是否运行在WSL中
func IsInWSL() bool {
	if f, err := os.Open("/proc/version"); err != nil {
		return false
	} else {
		defer f.Close()
		if content, err := ioutil.ReadAll(f); err != nil {
			return false
		} else {
			return strings.Contains(strings.ToLower(string(content)), "microsoft")
		}
	}

	return false
}

// IsInWSL2 是否运行在WSL2中
func IsInWSL2() bool {
	if f, err := os.Open("/proc/version"); err != nil {
		return false
	} else {
		defer f.Close()
		if content, err := ioutil.ReadAll(f); err != nil {
			return false
		} else {
			return strings.Contains(strings.ToLower(string(content)), "wsl2")
		}
	}

	return false
}
