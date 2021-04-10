package utils

import (
	"fmt"
	"reflect"
)

// functionCache keeps genericFunc reflection objects in cache.
type functionCache struct {
	FnValue  reflect.Value
	FnType   reflect.Type
	TypesIn  []reflect.Type
	TypesOut []reflect.Type
}

// genericFunc is a type used to validate and call dynamic functions.
type genericFunc struct {
	Cache *functionCache
}

// Call calls a dynamic function.
func (g *genericFunc) Call(params ...interface{}) interface{} {
	paramsIn := make([]reflect.Value, len(params))
	for i, param := range params {
		paramsIn[i] = reflect.ValueOf(param)
	}
	paramsOut := g.Cache.FnValue.Call(paramsIn)
	if len(paramsOut) >= 1 {
		return paramsOut[0].Interface()
	}
	return nil
}

func NewGenericFunc(fn interface{}) (*genericFunc, error) {
	cache := &functionCache{}
	cache.FnValue = reflect.ValueOf(fn)

	if cache.FnValue.Type().Kind() != reflect.Func {
		return nil, fmt.Errorf("is not a function type. It is a '%s'", cache.FnValue.Type())
	}
	cache.FnType = cache.FnValue.Type()
	numTypesIn := cache.FnType.NumIn()
	cache.TypesIn = make([]reflect.Type, numTypesIn)
	for i := 0; i < numTypesIn; i++ {
		cache.TypesIn[i] = cache.FnType.In(i)
	}

	numTypesOut := cache.FnType.NumOut()
	cache.TypesOut = make([]reflect.Type, numTypesOut)
	for i := 0; i < numTypesOut; i++ {
		cache.TypesOut[i] = cache.FnType.Out(i)
	}

	return &genericFunc{Cache: cache}, nil
}

func Invoke(fn interface{}, args ...interface{}) interface{} {
	if _fn, err := NewGenericFunc(fn); err == nil {
		return _fn.Call(args...)
	}
	return nil
}
