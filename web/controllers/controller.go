package controllers

import (
	"github.com/gin-gonic/gin"
	"time"
)

type Controller struct {
	Context *gin.Context
}

type IController interface {
	SuccessResponse(code, data interface{})
	ErrorResponse(exception IResponseException, data interface{})
}

// ErrorResponse default error response
func (c *Controller) ErrorResponse(exception IResponseException, data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.Abort()
	c.Context.JSON(exception.GetStatusCode(), Result{
		Code:     exception.GetCode(),
		Message:  exception.GetMessage(),
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// SuccessResponse default success response
func (c *Controller) SuccessResponse(code, data interface{}) {
	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(200, Result{
		Code:     code,
		Message:  "",
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

func (c *Controller) shouldBindQuery() {

}
