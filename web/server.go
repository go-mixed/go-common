package web

import (
	"context"
	"crypto/tls"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/http"
	list_utils "gopkg.in/go-mixed/go-common.v1/utils/list"
	text_utils "gopkg.in/go-mixed/go-common.v1/utils/text"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DomainConfig struct {
	domain  string
	handler http.Handler
}

type HttpServerOptions struct {
	Host              string
	IdleTimeout       time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	MaxHeaderBytes    int
	ReadHeaderTimeout time.Duration

	// 是否支持 Http2 without TLS
	UseH2C bool
}

type HttpServer struct {
	*HttpServerOptions

	orderedDomainConfigs []*DomainConfig
	middleware           *MiddlewarePipeline
	certs                []*Certificate
	mu                   sync.Mutex
	logger               utils.ILogger
	// 为了加快命中, 当有访问时，会将请求域名所指向的handler缓存到lru中
	domainCache *lru.TwoQueueCache
	// 内部运行的http.Server
	nativeHttpServer *http.Server
	mainCtxCancel    context.CancelFunc
}

func NewHttpServer(httpServerOptions *HttpServerOptions) *HttpServer {
	cache, _ := lru.New2Q(128)
	s := &HttpServer{
		HttpServerOptions:    httpServerOptions,
		orderedDomainConfigs: make([]*DomainConfig, 0, 1),
		mu:                   sync.Mutex{},
		logger:               utils.NewDefaultLogger(),
		domainCache:          cache,
	}
	s.middleware = NewMiddlewarePipeline(s.controllerHandlerFunc())
	return s
}

func DefaultServerOptions(host string) *HttpServerOptions {
	return &HttpServerOptions{
		Host:        host,
		IdleTimeout: 10 * time.Second,
	}
}

func (c *Certificate) CertFileInfo() os.FileInfo {
	if c.certFileInfo == nil {
		c.certFileInfo, _ = os.Stat(c.CertFile)
	}
	return c.certFileInfo
}

func (c *Certificate) KeyFileInfo() os.FileInfo {
	if c.keyFileInfo == nil {
		c.keyFileInfo, _ = os.Stat(c.KeyFile)
	}
	return c.keyFileInfo
}

func (c *HttpServer) HasDefaultDomain() bool {
	return c.ContainsDomain("*")
}

// ContainsDomain 是否包含此域名, 此函数是判断完全相等, 如果需要匹配通配符, 使用 MatchDomain
func (c *HttpServer) ContainsDomain(domain string) bool {
	return list_utils.Find(c.orderedDomainConfigs, func(value *DomainConfig) bool {
		return strings.EqualFold(value.domain, domain)
	}) >= 0
}

// ContainsCert 是否包含此证书, 需要cert/key都相等
func (c *HttpServer) ContainsCert(cert *Certificate) bool {
	return list_utils.Find(c.certs, func(value *Certificate) bool {
		return os.SameFile(cert.CertFileInfo(), value.CertFileInfo()) && os.SameFile(cert.KeyFileInfo(), value.KeyFileInfo())
	}) >= 0
}

// AddServeHandler 添加域名, serveHTTP, 证书
// 可以使用 AddCertificate 传递证书
// 注意: 如果有传递证书，证书DNS Name必须包含所传递的domains（此函数并不检查），不然，需要分多次添加
func (c *HttpServer) AddServeHandler(domains http_utils.Domains, handler http.Handler, cert *Certificate) error {
	if err := c.AddCertificate(cert); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 添加
	domainConfigs := c.orderedDomainConfigs[:]
	for _, domain := range domains {
		domain = strings.ToLower(domain)
		if c.ContainsDomain(domain) {
			return errors.Errorf("domain duplicate: \"%s\" is already in the server config", domain)
		}

		domainConfigs = append(domainConfigs, &DomainConfig{
			domain:  domain,
			handler: handler,
		})
	}

	// 按照域名的特有方式进行排序
	http_utils.SortDomains(domainConfigs, func(v *DomainConfig) string {
		return v.domain
	})

	c.ClearDomainCache()

	c.orderedDomainConfigs = domainConfigs
	return nil
}

// Use 添加中间件
//
//		这种嵌套方式的中间件, 可以运行在controller前, 也可以运行在controller后
//	  - 运行在 controller 前，一般为修改request的数据
//	    Use(func(w, r, next) {
//	    r.Path = "/abc" + r.Path // 修改path
//	    next.ServeHTTP(w, r)
//	    }
//	  - 运行在 controller 后，一般为修改response的数据
//	    Use(func(w, r, next) {
//	    next.ServeHTTP(w, r) // 先运行controller
//	    w.Write(...)
//	    }
func (c *HttpServer) Use(fn ...Middleware) *HttpServer {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.middleware.Push(fn...)

	return c
}

func (c *HttpServer) SetLogger(logger utils.ILogger) *HttpServer {
	c.logger = logger
	return c
}

func (c *HttpServer) AddCertificate(certs ...*Certificate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cert := range certs {
		if cert == nil {
			continue
		}
		if cert.CertFileInfo() == nil || cert.KeyFileInfo() == nil {
			return errors.Errorf("cert \"%s\" or key \"%s\" is invalid", cert.CertFile, cert.KeyFile)
		}
		if !c.ContainsCert(cert) { // 添加cert列表
			c.certs = append(c.certs, cert)
		}
	}
	return nil
}

func (c *HttpServer) RemoveDomainHandler(domain string) *HttpServer {
	c.mu.Lock()
	defer c.mu.Unlock()

	domain = strings.ToLower(domain)
	for i, domainConfig := range c.orderedDomainConfigs {
		if domainConfig.domain == domain {
			c.orderedDomainConfigs = append(c.orderedDomainConfigs[0:i], c.orderedDomainConfigs[i+1:]...)
			return c
		}
	}
	return c
}

// ClearDomainCache 清理域名的匹配结果
func (c *HttpServer) ClearDomainCache() *HttpServer {
	c.domainCache.Purge()
	return c
}

// SetDefaultServeHandler 设置默认的ServeHandler, 即添加一个通配符*的域名
func (c *HttpServer) SetDefaultServeHandler(handler http.Handler, cert *Certificate) error {
	return c.AddServeHandler([]string{"*"}, handler, cert)
}

// GetHost 获取待监听的HOST, 比如: 0.0.0.0:80
func (c *HttpServer) GetHost() string {
	return c.Host
}

// IsTLS 是否提供TLS服务
func (c *HttpServer) IsTLS() bool {
	return len(c.GetCertificates()) > 0
}

// GetCertificates 获取证书列表
func (c *HttpServer) GetCertificates() []*Certificate {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.certs
}

func (c *HttpServer) rememberDomainCache(key string, callback func() *DomainConfig) *DomainConfig {
	if val, ok := c.domainCache.Get(key); ok {
		if val == nil {
			return nil
		}
		return val.(*DomainConfig)
	}

	_val := callback()
	c.domainCache.Add(key, _val)
	return _val
}

// MatchDomain 匹配域名, 使用通配符的方式，如果只添加了一个domain，则不论是否匹配都返回该serveHandler
func (c *HttpServer) MatchDomain(domain string) *DomainConfig {
	// 只有一个的情况快速返回
	if len(c.orderedDomainConfigs) == 1 && c.orderedDomainConfigs[0].domain == "*" {
		return c.orderedDomainConfigs[0]
	}

	// 用Lru cache做持久保存，如果域名解绑，需要自行清理cache
	return c.rememberDomainCache(fmt.Sprintf("domain-handlerStack-%s", domain), func() *DomainConfig {
		c.mu.Lock()
		defer c.mu.Unlock()
		for _, domainConfig := range c.orderedDomainConfigs {
			if text_utils.WildcardMatch(domainConfig.domain, domain) {
				return domainConfig
			}
		}
		return nil
	})
}

func (c *HttpServer) controllerHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := http_utils.DomainFromRequestHost(r.Host)
		if domainConfig := c.MatchDomain(domain); domainConfig != nil {
			domainConfig.handler.ServeHTTP(w, r)
		}
	}
}

// ServeHTTP HTTP入口函数，执行中间件后，再匹配域名
func (c *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.middleware.Copy().ServeHTTP(w, r)
}

func (c *HttpServer) BuildServer() *http.Server {
	var handler http.Handler = c

	// 启用H2c
	if !c.UseH2C {
		h2s := &http2.Server{}
		handler = h2c.NewHandler(c, h2s)
	}

	return &http.Server{
		Addr:              c.GetHost(),
		IdleTimeout:       c.IdleTimeout,
		ReadTimeout:       c.ReadTimeout,
		WriteTimeout:      c.WriteTimeout,
		MaxHeaderBytes:    c.MaxHeaderBytes,
		ReadHeaderTimeout: c.ReadHeaderTimeout,
		Handler:           handler,
	}
}

// Run 对外主函数, 用于运行http(s) server
// 关闭可以通过 Close(), 或者Ctrl+C之类的进程结束信号, 或者传入ctx来控制
// 停止后, 可以再次运行。
func (c *HttpServer) Run(ctx context.Context, configServerFunc func(server *http.Server) error) error {

	if c.nativeHttpServer != nil {
		return errors.New("server is already running")
	}

	c.nativeHttpServer = c.BuildServer()

	_, c.mainCtxCancel = c.listenContext(ctx)

	// 启动http server
	if !c.IsTLS() {

		if configServerFunc != nil {
			if err := configServerFunc(c.nativeHttpServer); err != nil {
				return err
			}
		}

		c.logger.Infof("Start http server on %s", c.GetHost())

		if err := c.nativeHttpServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				c.logger.Info("http server closed")
			} else {
				return err
			}
		}
	} else { // 启动https server
		//
		if err := SetServerTLSCerts(c.nativeHttpServer, c.GetCertificates()); err != nil {
			return err
		}

		if configServerFunc != nil {
			if err := configServerFunc(c.nativeHttpServer); err != nil {
				return err
			}
		}

		c.logger.Infof("Start https server on %s", c.GetHost())

		if err := c.nativeHttpServer.ListenAndServeTLS("", ""); err != nil {
			if err == http.ErrServerClosed {
				c.logger.Info("https server closed")
			} else {
				return err
			}
		}
	}

	return nil
}

// Close 关闭服务器
func (c *HttpServer) Close() error {
	if c.mainCtxCancel != nil {
		// 释放listenContext中阻塞的协程
		c.mainCtxCancel()
		c.mainCtxCancel = nil
	}

	if c.nativeHttpServer != nil {
		if err := c.nativeHttpServer.Close(); err != nil {
			c.logger.Fatalf("Server closed: %s", err.Error())
			return err
		}
		c.nativeHttpServer = nil
	}

	return nil
}

// 监听停止信号, ctx为nil时仅仅收听进程退出信号
// 返回一对新的ctx, 用于退出函数内的协程
func (c *HttpServer) listenContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// 新建这个context是为了在主动执行Close的情况下, 让下面的协程能正常退出
	runningCtx, runningCancel := context.WithCancel(context.Background())
	if ctx == nil { // 为空则赋值一个默认context
		ctx = context.Background()
	}

	//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
	exitSign := make(chan os.Signal)
	signal.Notify(exitSign, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		defer close(exitSign)
		defer runningCancel() // 避免泄露

		select {
		case <-exitSign: // 退出信号
			c.logger.Info("exit signal of process received.")
			_ = c.Close()
		case <-ctx.Done(): // 监控到ctx退出
			_ = c.Close()
		case <-runningCtx.Done(): // 这里被是被 Close 触发, 只是为了退出本协程, 避免协程一直阻塞
		}

	}()

	return runningCtx, runningCancel
}

// SetServerTLSCerts 给http.Server{}一次添加多个TLS证书
func SetServerTLSCerts(srv *http.Server, certs []*Certificate) error {
	var err error

	if srv.TLSConfig == nil {
		srv.TLSConfig = &tls.Config{}
	}

	srv.TLSConfig.Certificates = make([]tls.Certificate, len(certs))
	for i, v := range certs {
		srv.TLSConfig.Certificates[i], err = tls.LoadX509KeyPair(v.CertFile, v.KeyFile)
		if err != nil {
			return err
		}
	}

	return nil

}
