package utils

import (
	"strconv"
	"strings"
)

func IsInt(val string) bool {
	_, err := strconv.Atoi(val)
	return err == nil
}

func IsInt64(val string) bool {
	_, err := strconv.ParseInt(val, 10, 64)
	return err == nil
}

func ParseInt(s string, base int, bitSize int, _default int64) int64 {
	if i, err := strconv.ParseInt(s, base, bitSize); err == nil {
		return i
	}

	return _default
}

func ParseFloat(s string, bitSize int, _default float64) float64 {
	if f, err := strconv.ParseFloat(s, bitSize); err == nil {
		return f
	}
	return _default
}

func Atoi(s string, _default int) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	return _default
}

func Atof(s string, _default float32) float32 {
	return float32(ParseFloat(s, 32, float64(_default)))
}

func Atof64(s string, _default float64) float64 {
	return ParseFloat(s, 32, _default)
}

func Atoi64(s string, _default int64) int64 {
	return ParseInt(s, 10, 64, _default)
}

func Itoa(i int) string {
	return strconv.Itoa(i)
}

func I64toa(i int64) string {
	return strconv.FormatInt(i, 10)
}

// PercentageToFloat XX.xx% => 0.XXxx
func PercentageToFloat(p string) float32 {
	return Atof(strings.TrimSuffix(p, "%"), 0) / 100
}
