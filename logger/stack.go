package logger

import (
	"fmt"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func stackToLines(stack stackTracer) []string {
	var lines []string
	for _, f := range stack.StackTrace() {
		lines = append(lines, fmt.Sprintf("%+s:%d", f, f))
	}

	return lines
}
