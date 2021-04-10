package web

import (
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go-common/utils"
	"net/http"
	"path/filepath"
	"time"
)

type ServerOptions struct {
	Debug bool
	Host  string
	Cert  string
	Key   string
}

func NewGinEngine(options *ServerOptions) *gin.Engine {
	logger := utils.GetLogger()

	router := gin.Default()
	router.Use(ginzap.Ginzap(logger, time.RFC3339, true)) // 使用zap作为logger
	router.Use(func(context *gin.Context) {
		context.Set("request_at", time.Now())
	}) // 注册当前时间
	pprof.Register(router) // 注册火焰图pprof

	// 注册模板
	router.LoadHTMLGlob(filepath.Join(utils.GetCurrentDir(), "resources/views/**/*"))
	// 注册静态文件夹
	router.Static("/assets", filepath.Join(utils.GetCurrentDir(), "resources/assets"))
	router.StaticFile("/favicon.ico", filepath.Join(utils.GetCurrentDir(), "resources/assets/img/favicon.ico"))

	return router
}

func RunServer(options *ServerOptions, router *gin.Engine, stopChan <-chan bool) error {

	if !options.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	sugarLogger := utils.GetSugaredLogger()

	server := &http.Server{
		Addr:    options.Host,
		Handler: router,
	}

	go func() {
		<-stopChan

		if err := server.Close(); err != nil {
			sugarLogger.Fatal("Server Close: ", err)
		}
	}()

	// 启动http server
	if options.Cert == "" || options.Key == "" {
		sugarLogger.Infof("Start http server on %s", options.Host)

		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				sugarLogger.Info("http server closed")
			} else {
				return err
			}
		}
	} else { // 启动https server
		sugarLogger.Infof("Start https server on %s", options.Host)

		if err := server.ListenAndServeTLS(options.Cert, options.Key); err != nil {
			if err == http.ErrServerClosed {
				sugarLogger.Info("https server closed")
			} else {
				return err
			}
		}
	}

	return nil
}
