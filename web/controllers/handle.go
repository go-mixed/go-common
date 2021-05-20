package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"net/http"
	"reflect"
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

func ControllerHandle(controllerName, methodName string) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		controller, err := NewController(controllerName, ctx)
		if err != nil {
			ctx.AbortWithStatus(http.StatusNotFound)
			_, _ = ctx.Writer.WriteString(err.Error())
		} else if !utils.HasMethod(controller, methodName) {
			controller.ErrorResponse(NewResponseException(http.StatusNotFound, http.StatusNotFound, fmt.Sprintf("controller method [%s@%s] not founud", controllerName, methodName)), nil)
		} else if res, err := callControllerMethod(controller, methodName); err == nil {
			controller.SuccessResponse(0, res)
		} else {
			controller.ErrorResponse(err, res)
		}
	}
}

func callControllerMethod(controller IController, methodName string, args ...interface{}) (interface{}, error) {
	res, err := utils.CallMethod2(controller, methodName, args...)

	// format string/error/any pointer to error
	errValueOf := reflect.ValueOf(err)
	errKind := errValueOf.Kind()
	if errKind == reflect.Ptr && !errValueOf.IsNil() {
		switch err.(type) {
		case error:
			return res, err.(error)
		default:
			return res, fmt.Errorf("%#v", err)
		}
	} else if errKind == reflect.String && err != "" {
		return res, fmt.Errorf("%#v", err.(string))
	}

	return res, nil
}
