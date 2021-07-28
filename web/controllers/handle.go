package controllers

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go-common/utils/core"
	"net/http"
)

var controllerRegistry map[string]func(ctx *gin.Context) IController

func init() {
	controllerRegistry = map[string]func(ctx *gin.Context) IController{}
}

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
		} else if !core_utils.HasMethod(controller, methodName) {
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
			if res, err := after(controller, r, e); err == nil {
				controller.SuccessResponse(0, res)
			} else {
				controller.ErrorResponse(err, res)
			}
		}

	}
}

func callControllerMethod(controller IController, methodName string, args ...interface{}) (interface{}, error) {
	res, err := core_utils.CallMethod2(controller, methodName, args...)

	if !core_utils.IsInterfaceNil(err) {
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
