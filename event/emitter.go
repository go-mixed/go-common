package event

import (
	"context"
	"github.com/olebedev/emitter"
	"go-common/utils/core"
)

type Emitter struct {
	*emitter.Emitter
	ctxCancel context.CancelFunc
	listeners map[string]interface{}
}

func NewEmitter(cap uint) *Emitter {
	return &Emitter{
		Emitter:   emitter.New(cap),
		listeners: map[string]interface{}{},
	}
}

func (e *Emitter) Subscribe(topic string, handler interface{}) {
	e.listeners[topic] = handler
}

func (e *Emitter) Stop() {
	if e.ctxCancel != nil {
		e.ctxCancel()
	}
	e.ctxCancel = nil
}

func (e *Emitter) RunConsumer(ctx context.Context) {
	ctx1, cancel := context.WithCancel(ctx)
	defer e.Stop()

	e.ctxCancel = cancel
	go func() {
		core.WaitForStopped(ctx1.Done())
		e.Off("*")
	}()

	for event := range e.On("*") {
		if handler, ok := e.listeners[event.OriginalTopic]; ok {
			_ = core.Invoke(handler, event.Args...)
		}
	}
}
