package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"reflect"
	"time"
)

type Controller struct {
	Context *gin.Context
}

type ControllerInterface interface {
	JsonSuccessResponse(data interface{})
	JsonErrorResponse(code int, message string, data interface{})
	CallMethod(controller ControllerInterface, method string, args ...interface{}) (interface{}, *ResponseException)
}

func (c *Controller) JsonErrorResponse(code int, message string, data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(400, Result{
		Code:     code,
		Message:  message,
		Data:     data,
		Duration: float64(duration) / float64(time.Second),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

func (c *Controller) JsonSuccessResponse(data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(200, Result{
		Code:     0,
		Message:  "",
		Data:     data,
		Duration: float64(duration) / float64(time.Second),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

func (c *Controller) CallMethod(controller ControllerInterface, method string, args ...interface{}) (interface{}, *ResponseException) {
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

func (c *Controller) shouldBindQuery() {

}
