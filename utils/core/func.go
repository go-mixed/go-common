package core

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"runtime"
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
func (g *genericFunc) Call(params ...any) []any {
	paramsIn := make([]reflect.Value, len(params))
	for i, param := range params {
		paramsIn[i] = reflect.ValueOf(param)
	}
	paramsOut := g.Cache.FnValue.Call(paramsIn)
	if len(paramsOut) >= 1 {
		res := make([]any, len(paramsOut))
		for i := 0; i < len(paramsOut); i++ {
			res[i] = paramsOut[i].Interface()
		}
		return res
	}
	return nil
}

func NewGenericFunc(fn any) (*genericFunc, error) {
	cache := &functionCache{}
	cache.FnValue = reflect.ValueOf(fn)

	if cache.FnValue.Type().Kind() != reflect.Func {
		return nil, errors.Errorf("is not a function type. It is a '%s'", cache.FnValue.Type())
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

func NewInstanceFunc(instance any, method string) (*genericFunc, error) {
	vOf := reflect.ValueOf(instance)
	if _, ok := vOf.Type().MethodByName(method); ok {
		return NewGenericFunc(vOf.MethodByName(method).Interface())
	} else {
		return nil, errors.Errorf("method %s not found", method)
	}
}

// NewElemTypeSlice creates a slice of items elem types.
func NewElemTypeSlice(items ...any) []reflect.Type {
	typeList := make([]reflect.Type, len(items))
	for i, item := range items {
		typeItem := reflect.TypeOf(item)
		if typeItem.Kind() == reflect.Ptr {
			typeList[i] = typeItem.Elem()
		}
	}
	return typeList
}

func HasMethod(instance any, method string) bool {
	_, ok := reflect.TypeOf(instance).MethodByName(method)
	return ok
}

func CallMethod(instance any, method string, args ...any) any {
	if _fn, err := NewInstanceFunc(instance, method); err == nil {
		if res := _fn.Call(args...); len(res) >= 1 {
			return res[0]
		}
	} else {
		fmt.Printf("call method %s fail: %s", method, err.Error())
	}
	return nil
}

func CallMethod2(instance any, method string, args ...any) (any, any) {
	if _fn, err := NewInstanceFunc(instance, method); err == nil {
		if res := _fn.Call(args...); len(res) >= 2 {
			return res[0], res[1]
		}
	} else {
		fmt.Printf("call method2 %s fail: %s", method, err.Error())
	}
	return nil, nil
}

func CallMethod3(instance any, method string, args ...any) (any, any, any) {
	if _fn, err := NewInstanceFunc(instance, method); err == nil {
		if res := _fn.Call(args...); len(res) >= 3 {
			return res[0], res[1], res[2]
		}
	} else {
		fmt.Printf("call method3 %s fail: %s", method, err.Error())
	}
	return nil, nil, nil
}

func Invoke(fn any, args ...any) any {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 1 {
			return res[0]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil
}

func Invoke2(fn any, args ...any) (any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 2 {
			return res[0], res[1]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil
}

func Invoke3(fn any, args ...any) (any, any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 3 {
			return res[0], res[1], res[2]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil
}

func Invoke4(fn any, args ...any) (any, any, any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 4 {
			return res[0], res[1], res[2], res[3]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil
}

func Invoke5(fn any, args ...any) (any, any, any, any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 5 {
			return res[0], res[1], res[2], res[3], res[4]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil, nil
}

func Invoke6(fn any, args ...any) (any, any, any, any, any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 6 {
			return res[0], res[1], res[2], res[3], res[4], res[5]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil, nil, nil
}

func Invoke7(fn any, args ...any) (any, any, any, any, any, any, any) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 7 {
			return res[0], res[1], res[2], res[3], res[4], res[5], res[6]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil, nil, nil, nil
}

func Invoke8(fn any, args ...any) (any, any, interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 8 {
			return res[0], res[1], res[2], res[3], res[4], res[5], res[6], res[7]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil, nil, nil, nil, nil
}
func Invoke9(fn interface{}, args ...interface{}) (interface{}, interface{}, interface{}, interface{}, interface{}, interface{}, interface{}, interface{}, interface{}) {
	if _fn, err := NewGenericFunc(fn); err == nil {
		if res := _fn.Call(args...); len(res) >= 9 {
			return res[0], res[1], res[2], res[3], res[4], res[5], res[6], res[7], res[8]
		}
	} else {
		fmt.Printf("invoke fail: %s", err.Error())
	}
	return nil, nil, nil, nil, nil, nil, nil, nil, nil
}

// NameOfFunction
func NameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
