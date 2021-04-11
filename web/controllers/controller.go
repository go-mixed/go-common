package controllers

import (
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"time"
)

type Controller struct {
	Context *gin.Context
}

type ControllerInterface interface {
	JsonSuccessResponse(data interface{})
	JsonErrorResponse(code int, message string, data interface{})
}

func (c *Controller) JsonErrorResponse(code int, message string, data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	statusCode := utils.If(code >= 400 && code <= 599, code, 400).(int)

	c.Context.JSON(statusCode, Result{
		Code:     code,
		Message:  message,
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

func (c *Controller) JsonSuccessResponse(data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(200, Result{
		Code:     0,
		Message:  "",
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

func (c *Controller) shouldBindQuery() {

}
