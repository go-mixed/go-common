package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"io"
)

// CustomRender 表示该Controller的方法是自定义渲染, 不需调用ErrorResponse/ApiResponse
var CustomRender = errors.New("custom render")

type ControllerMethod[T any] func(ctx *gin.Context) (T, error)

func Handle[T any](controller IController, controllerMethod ControllerMethod[T]) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		res, err := controllerMethod(ctx)
		if err != nil {
			if !errors.Is(err, CustomRender) {
				controller.withContext(ctx).ErrorResponse(err, res)
			}
		} else {
			controller.withContext(ctx).ApiResponse(0, res)
		}
	}
}

// DiscardBody 丢弃Body内容, golang的http server必须要读取，不然会断开连接
func DiscardBody(ctx *gin.Context) error {
	if ctx.Request.Body != nil {
		// 必须要读取完毕, 不然会断开连接 https://github.com/golang/go/issues/23262
		_, err := io.Copy(io.Discard, ctx.Request.Body)
		if err != nil {
			return err
		}
		return ctx.Request.Body.Close()
	}
	return nil
}
