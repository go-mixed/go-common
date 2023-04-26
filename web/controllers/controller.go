package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"io"
	"net/http"
	"time"
)

type Controller struct {
	ctx *gin.Context
}

type IController interface {
	withContext(ctx *gin.Context) IController
	ApiResponse(code, data any)
	ErrorResponse(err error, data any)
}

var _ IController = &Controller{}

func (c *Controller) JsonCheck(d any) error {
	if err := c.ctx.ShouldBindJSON(&d); err != nil {
		if err == io.EOF {
			return errors.Errorf("empty body, must be a json")
		}
		return err
	}
	return nil
}

func (c *Controller) withContext(ctx *gin.Context) IController {
	return &Controller{
		ctx: ctx,
	}
}

// ErrorResponse default error response
func (c *Controller) ErrorResponse(err error, data any) {

	duration := time.Now().Sub(c.ctx.GetTime("request_at"))

	_err := c.EnsureErrorResponse(err)

	c.ctx.Abort()
	c.ctx.JSON(_err.GetStatusCode(), utils.Result{
		Code:     _err.GetCode(),
		Message:  _err.GetMessage(),
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// ApiResponse default success response
func (c *Controller) ApiResponse(code, data any) {

	duration := time.Now().Sub(c.ctx.GetTime("request_at"))

	c.ctx.JSON(200, utils.Result{
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

func (c *Controller) DiscardBody() error {
	return DiscardBody(c.ctx)
}
