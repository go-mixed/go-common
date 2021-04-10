package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"reflect"
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

		if res, err := callControllerMethod(controller, methodName); err == nil {
			controller.JsonSuccessResponse(res)
		} else {
			controller.JsonErrorResponse(err.Code, err.Message, res)
		}
	}
}

func callControllerMethod(controller ControllerInterface, method string, args ...interface{}) (interface{}, *ResponseException) {
	res, err := utils.CallMethod2(controller, method, args...)

	errKind := reflect.ValueOf(err).Kind()
	if errKind == reflect.Ptr && !reflect.ValueOf(err).IsNil() {
		switch reflect.TypeOf(err).Elem().Name() {
		case "errorString":
			return res, NewResponseException(-1, err.(error).Error())
		case "ResponseException":
			return res, err.(*ResponseException)
		default:
			return res, NewResponseException(-1, fmt.Sprintf("%#v", err))
		}
	} else if errKind == reflect.String && err != "" {
		return res, NewResponseException(-1, err.(string))
	}

	return res, nil
}
