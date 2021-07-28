package web

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"go-common/utils/io"
	"path/filepath"
	"time"
)

type GinOptions struct {
	Debug            bool
	RegisterPprof    bool
	HTMLTemplatePath string
	StaticPath       string
	StaticFiles      map[string]string
}

func DefaultGinOptions(debug, registerPprof bool) *GinOptions {
	currentDir := io_utils.GetCurrentDir()
	return &GinOptions{
		Debug:            debug,
		RegisterPprof:    registerPprof,
		HTMLTemplatePath: filepath.Join(currentDir, "resources/views/**/*"),
		StaticPath:       filepath.Join(currentDir, "resources/assets"),
		StaticFiles: map[string]string{
			"/favicon.ico": filepath.Join(currentDir, "resources/assets/img/favicon.ico"),
		},
	}
}

func NewGinEngine(options *GinOptions) *gin.Engine {
	logger := utils.GetLogger()

	router := gin.Default()
	if !options.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router.Use(GinZap(logger, time.RFC3339, true)) // 使用zap作为logger
	router.Use(func(context *gin.Context) {
		context.Set("request_at", time.Now())
	}) // 注册当前时间

	router.Use(gin.Recovery())

	if options.RegisterPprof {
		pprof.Register(router) // 注册火焰图pprof
	}

	// 注册模板
	if options.HTMLTemplatePath != "" {
		router.LoadHTMLGlob(options.HTMLTemplatePath)
	}

	// 注册静态文件夹
	if options.StaticPath != "" {
		router.Static("/assets", options.StaticPath)
	}

	// 注册静态文件
	for file, path := range options.StaticFiles {
		router.StaticFile(file, path)
	}

	return router
}
