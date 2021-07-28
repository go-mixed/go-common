package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"go-common/utils/core"
	"net/http"
	"time"
)

type Controller struct {
	Context *gin.Context
}

type IController interface {
	SuccessResponse(code, data interface{})
	ErrorResponse(err error, data interface{})
}

func (c *Controller) Render(data render.Render) error {
	_render := data.(render.Render)
	_render.WriteContentType(c.Context.Writer)
	return data.(render.Render).Render(c.Context.Writer)
}

// ErrorResponse default error response
func (c *Controller) ErrorResponse(err error, data interface{}) {
	if !core_utils.IsInterfaceNil(data) {
		switch data.(type) {
		case render.Render:
			c.Render(data.(render.Render))
			return
		}
	}

	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	_err := c.EnsureErrorResponse(err)

	c.Context.Abort()
	c.Context.JSON(_err.GetStatusCode(), Result{
		Code:     _err.GetCode(),
		Message:  _err.GetMessage(),
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// SuccessResponse default success response
func (c *Controller) SuccessResponse(code, data interface{}) {
	if !core_utils.IsInterfaceNil(data) {
		switch data.(type) {
		case render.Render:
			c.Render(data.(render.Render))
			return
		}
	}

	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(200, Result{
		Code:     code,
		Message:  "",
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// EnsureErrorResponse turn error to IResponseException
func (c *Controller) EnsureErrorResponse(err error) IResponseException {
	var _err IResponseException
	switch err.(type) {
	case IResponseException:
		_err = err.(IResponseException)
	default:
		_err = NewResponseException(-1, http.StatusBadRequest, err.Error())
	}
	return _err
}
