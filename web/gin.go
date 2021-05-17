package web

import (
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"path/filepath"
	"time"
)

func NewGinEngine(debug bool, withPprof bool) *gin.Engine {
	logger := utils.GetLogger()

	router := gin.Default()
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true)) // 使用zap作为logger
	router.Use(func(context *gin.Context) {
		context.Set("request_at", time.Now())
	}) // 注册当前时间
	if withPprof {
		pprof.Register(router) // 注册火焰图pprof
	}

	// 注册模板
	router.LoadHTMLGlob(filepath.Join(utils.GetCurrentDir(), "resources/views/**/*"))
	// 注册静态文件夹
	router.Static("/assets", filepath.Join(utils.GetCurrentDir(), "resources/assets"))
	router.StaticFile("/favicon.ico", filepath.Join(utils.GetCurrentDir(), "resources/assets/img/favicon.ico"))

	return router
}