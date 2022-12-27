package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"go-common/utils"
	"io"
	"net/http"
	"time"
)

type Controller struct {
	Context *gin.Context
}

type IController interface {
	SuccessResponse(code, data any)
	ErrorResponse(err error, data any)
}

func (c *Controller) JsonCheck(d any) error {
	if err := c.Context.ShouldBindJSON(&d); err != nil {
		if err == io.EOF {
			return errors.Errorf("empty body, must be a json")
		}
		return err
	}
	return nil
}

// ErrorResponse default error response
func (c *Controller) ErrorResponse(err error, data any) {

	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	_err := c.EnsureErrorResponse(err)

	c.Context.Abort()
	c.Context.JSON(_err.GetStatusCode(), utils.Result{
		Code:     _err.GetCode(),
		Message:  _err.GetMessage(),
		Data:     data,
		Duration: float64(duration) / float64(time.Millisecond),
		At:       time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// SuccessResponse default success response
func (c *Controller) SuccessResponse(code, data any) {

	duration := time.Now().Sub(c.Context.GetTime("request_at"))

	c.Context.JSON(200, utils.Result{
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
	return DiscardBody(c.Context)
}
