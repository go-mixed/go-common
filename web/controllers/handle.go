package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go-common/utils/core"
	"io"
	"net/http"
)

var controllerRegistry map[string]func(ctx *gin.Context) IController

func init() {
	controllerRegistry = map[string]func(ctx *gin.Context) IController{}
}

// CustomRender 表示该Controller的方法是自定义渲染, 不需调用ErrorResponse/SuccessResponse
var CustomRender = errors.New("custom render")

func NewController(controllerName string, ctx *gin.Context) (IController, error) {
	if callback, ok := controllerRegistry[controllerName]; ok {
		if controller := callback(ctx); controller != nil {
			return controller, nil
		} else {
			return nil, fmt.Errorf("get nil controller [%s]", controllerName)
		}
	}

	return nil, fmt.Errorf("controller [%s] not exists", controllerName)
}

func RegisterController(controllerName string, fn func(ctx *gin.Context) IController) {
	controllerRegistry[controllerName] = fn
}

func ControllerHandler(controllerName, methodName string) gin.HandlerFunc {
	return ControllerHandlerFunc(controllerName, methodName, nil, nil)
}

func emptyBefore(IController) {

}

func emptyAfter(c IController, v interface{}, e error) (interface{}, error) {
	return v, e
}

func ControllerHandlerFunc(controllerName, methodName string, before func(IController), after func(IController, interface{}, error) (interface{}, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		controller, err := NewController(controllerName, ctx)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			_, _ = ctx.Writer.WriteString(err.Error())
		} else if !core.HasMethod(controller, methodName) {
			controller.ErrorResponse(NewResponseException(http.StatusNotFound, http.StatusNotFound, fmt.Sprintf("controller method [%s@%s] not founud", controllerName, methodName)), nil)
		} else {
			if before == nil {
				before = emptyBefore
			}
			if after == nil {
				after = emptyAfter
			}

			before(controller)
			r, e := callControllerMethod(controller, methodName)
			res, err := after(controller, r, e)

			if err == CustomRender {
				return
			} else if err == nil {
				controller.SuccessResponse(0, res)
			} else {
				controller.ErrorResponse(err, res)
			}
		}
	}
}

func callControllerMethod(controller IController, methodName string, args ...interface{}) (interface{}, error) {
	res, err := core.CallMethod2(controller, methodName, args...)

	if !core.IsInterfaceNil(err) {
		switch err.(type) {
		case error:
			return res, err.(error)
		case string:
			if err != "" {
				return res, errors.New(err.(string))
			}
		default:
			return res, fmt.Errorf("%#v", err)
		}
	}

	return res, nil
}

// DiscardBody 丢弃Body内容, golang的http server必须要读取，不然会断开连接
func DiscardBody(ctx *gin.Context) error {
	if ctx.Request.Body != nil {
		// 必须要读取完毕, 不然会断开连接 https://github.com/golang/go/issues/23262
		_, err := io.Copy(io.Discard, ctx.Request.Body)
		if err != nil {
			return err
		}
		return ctx.Request.Body.Close()
	}
	return nil
}
