package web

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/go-mixed/go-common.v1/utils/io"
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

func ApiGinOptions(debug, registerPprof bool) *GinOptions {
	return &GinOptions{
		Debug:         debug,
		RegisterPprof: registerPprof,
	}
}

func WebsiteGinOptions(debug, registerPprof bool) *GinOptions {
	currentDir := ioUtils.GetCurrentDir()
	return &GinOptions{
		Debug:            debug,
		RegisterPprof:    registerPprof,
		HTMLTemplatePath: filepath.Join(currentDir, "resources/views/**/*.html"),
		StaticPath:       filepath.Join(currentDir, "resources/assets"),
		StaticFiles: map[string]string{
			"/favicon.ico": filepath.Join(currentDir, "resources/assets/img/favicon.ico"),
		},
	}
}

func (o *GinOptions) NoResources() {
	o.HTMLTemplatePath = ""
	o.StaticPath = ""
	o.StaticFiles = map[string]string{}
}

func NewGinEngine(options *GinOptions, logger *zap.Logger) *gin.Engine {

	router := gin.New()

	if options.Debug {
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery()) // 捕获RecoveryWithZap无法捕获的错误
	if !options.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Add a ginzap middleware, which:
	//   - Logs all requests, like a combined access and error log.
	//   - Logs to stdout.
	//   - RFC3339 with UTC time format.
	router.Use(GinZap(logger, time.RFC3339, true)) // 使用zap作为logger

	router.Use(func(context *gin.Context) {
		context.Set("request_at", time.Now())
	}) // 注册当前时间

	// Logs all panic to error log
	//   - stack means whether output the stack info.
	router.Use(RecoveryWithZap(logger, true))

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
