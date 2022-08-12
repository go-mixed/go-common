package web

import (
	"container/list"
	"net/http"
)

type Middleware func(w http.ResponseWriter, r *http.Request, nextHandler http.Handler)

type MiddlewarePipeline struct {
	// 中间件链表
	pipeline *list.List
	// 真正的控制器的handle，在中间件链表末端运行
	controllerHandler http.Handler
	// 已运行到哪个中间件
	current *list.Element
	// 是否是副本
	isCopy bool
}

func NewMiddlewarePipeline(controllerHandler http.Handler) *MiddlewarePipeline {
	ls := list.New()

	return &MiddlewarePipeline{
		pipeline:          ls,
		controllerHandler: controllerHandler,
	}
}

func (m *MiddlewarePipeline) Push(fn ...Middleware) *MiddlewarePipeline {
	for _, n := range fn {
		m.pipeline.PushBack(n)
	}
	return m
}

func (m *MiddlewarePipeline) GetPipeline() *list.List {
	return m.pipeline
}

// Copy 每个http的会话都必须是单独一份copy, 即 middlewarePipeline.Copy().ServeHTTP(w, h)
//
// 返回http.Handler是为了隐藏Push方法，避免pipeline被修改
func (m *MiddlewarePipeline) Copy() http.Handler {
	return &MiddlewarePipeline{
		isCopy:            true,
		pipeline:          m.pipeline,
		controllerHandler: m.controllerHandler,
		current:           m.pipeline.Front(),
	}
}

func (m *MiddlewarePipeline) nextServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.current = m.current.Next()
	m.ServeHTTP(w, r)
}

// ServeHTTP 中间件执行入口, 注意：每个http会话必须运行在独立的MiddlewarePipeline副本，
// 即 middlewarePipeline.Copy().ServeHTTP(w, h)，
// 不然会引发异常
func (m *MiddlewarePipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.isCopy {
		panic("you must run middleware with Copy().ServeHTTP(w, r)")
	}
	if m.current != nil {
		if middleware, ok := m.current.Value.(Middleware); ok {
			middleware(w, r, http.HandlerFunc(m.nextServeHTTP))
		}
	} else { // 说明是结尾
		m.controllerHandler.ServeHTTP(w, r)
	}
}
