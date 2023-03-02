package task

import (
	"context"
	"time"
)

type State string

const (
	Prepare State = "prepare"
	Running State = "running"
	Timeout State = "timeout"
	Panic   State = "panic"
	Done    State = "done"
)

type Job struct {
	Callback func(ctx context.Context)
	Timeout  time.Duration
	State    State
	Error    error
	RunAt    time.Time
	FinishAt time.Time
}
