package event

import (
	"github.com/olebedev/emitter"
	"go-common/utils/core"
)

type Emitter struct {
	*emitter.Emitter
	stopChan chan struct{}
	listeners map[string]interface{}
}

func NewEmitter(cap uint) *Emitter {
	return &Emitter{
		Emitter: emitter.New(cap),
		stopChan: make(chan struct{}),
		listeners: map[string]interface{}{},
	}
}

func (e *Emitter) Subscribe(topic string, handler interface{})  {
	e.listeners[topic] = handler
}

func (e *Emitter) Stop() {
	close(e.stopChan)
}

func (e *Emitter) RunConsumer(stopChan <- chan struct{})  {

	go func() {
		core.WaitForStopped2(stopChan, e.stopChan)
		e.Off("*")
	}()

	for event := range e.On("*") {
		if handler, ok := e.listeners[event.OriginalTopic]; ok {
			_ = core.Invoke(handler, event.Args...)
		}
	}
}