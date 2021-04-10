package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

var controllerRegistry map[string]func(ctx *gin.Context) ControllerInterface

func init() {
	controllerRegistry = map[string]func(ctx *gin.Context) ControllerInterface{}
}

func NewController(controllerName string, ctx *gin.Context) ControllerInterface {
	if callback, ok := controllerRegistry[controllerName]; ok {
		return callback(ctx)
	} else {
		panic(fmt.Sprintf("controller [%s] not exists", controllerName))
	}
}

func RegisterController(controllerName string, fn func(ctx *gin.Context) ControllerInterface) {
	controllerRegistry[controllerName] = fn
}

func ControllerHandle(controllerName, methodName string) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		controller := NewController(controllerName, ctx)

		if res, err := controller.CallMethod(controller, methodName); err == nil {
			controller.JsonSuccessResponse(res)
		} else {
			controller.JsonErrorResponse(err.Code, err.Message, res)
		}
	}
}
