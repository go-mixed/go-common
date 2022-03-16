package web

import (
	"context"
	"crypto/tls"
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"go-common/utils"
	"go-common/utils/http"
	"go-common/utils/list"
	"go-common/utils/text"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Certificate struct {
	CertFile string `json:"cert"`
	KeyFile  string `json:"key"`

	certFileInfo os.FileInfo
	keyFileInfo  os.FileInfo
}

type Middleware func(w http.ResponseWriter, r *http.Request, nextHandler http.Handler)

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
}

type HttpServer struct {
	*HttpServerOptions

	orderedDomainConfigs []*DomainConfig
	middleware           []Middleware
	// 真正执行的handler入口
	handlerStack http.Handler
	certs        []*Certificate
	mu           sync.Mutex
	logger       utils.ILogger
	// 为了加快命中, 当有访问时，会将请求域名所指向的handler缓存到lru中
	domainCache *lru.TwoQueueCache
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
	s.handlerStack = s.defaultHandlerFunc() // 最后的handler
	return s
}

func NewCertificate(certFile, keyFile string) *Certificate {
	if certFile == "" || keyFile == "" {
		return nil
	}
	return &Certificate{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
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
			return fmt.Errorf("domain duplicate: \"%s\" is already in the server config", domain)
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
// 这种嵌套方式的中间件, 可以运行在controller前, 也可以运行在controller后
// 运行在 controller 前
// Use(func(w, r, next) {
// 		r.Path = "/abc" + r.Path // 修改path
//      next.ServeHTTP(w, r)
// }
// controller 后
// Use(func(w, r, next) {
//      next.ServeHTTP(w, r) // 先运行
// 		w.Write(...)
// }
//
func (c *HttpServer) Use(fn ...Middleware) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.middleware = append(c.middleware, fn...)

	// 倒着创建中间件管道
	// 嵌套顺序为
	// c.handlerStack = func m1(w, r) {
	//    func m2(w, r) {
	//        c.defaultHandlerFunc()(w, r)
	//    }(w, r)
	// }
	var last = c.defaultHandlerFunc()
	for i := len(c.middleware) - 1; i >= 0; i-- {
		func(_last http.HandlerFunc, m Middleware) {
			last = func(w http.ResponseWriter, r *http.Request) {
				m(w, r, _last)
			}
		}(last, c.middleware[i])
	}

	c.handlerStack = last
}

func (c *HttpServer) SetLogger(logger utils.ILogger) {
	c.logger = logger
}

func (c *HttpServer) AddCertificate(certs ...*Certificate) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cert := range certs {
		if cert == nil {
			continue
		}
		if cert.CertFileInfo() == nil || cert.KeyFileInfo() == nil {
			return fmt.Errorf("cert \"%s\" or key \"%s\" is invalid", cert.CertFile, cert.KeyFile)
		}
		if !c.ContainsCert(cert) { // 添加cert列表
			c.certs = append(c.certs, cert)
		}
	}
	return nil
}

func (c *HttpServer) RemoveDomainHandler(domain string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	domain = strings.ToLower(domain)
	for i, domainConfig := range c.orderedDomainConfigs {
		if domainConfig.domain == domain {
			c.orderedDomainConfigs = append(c.orderedDomainConfigs[0:i], c.orderedDomainConfigs[i+1:]...)
			return
		}
	}
}

// ClearDomainCache 清理域名的匹配结果
func (c *HttpServer) ClearDomainCache() {
	c.domainCache.Purge()
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

func (c *HttpServer) defaultHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		domain := http_utils.DomainFromRequestHost(r.Host)
		if domainConfig := c.MatchDomain(domain); domainConfig != nil {
			domainConfig.handler.ServeHTTP(w, r)
		}
	}
}

// ServeHTTP HTTP入口函数，匹配域名之后 分发到不同handler
func (c *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handlerStack.ServeHTTP(w, r)
}

func (c *HttpServer) BuildServer() *http.Server {
	return &http.Server{
		Addr:              c.GetHost(),
		IdleTimeout:       c.IdleTimeout,
		ReadTimeout:       c.ReadTimeout,
		WriteTimeout:      c.WriteTimeout,
		MaxHeaderBytes:    c.MaxHeaderBytes,
		ReadHeaderTimeout: c.ReadHeaderTimeout,
		Handler:           c,
	}
}

// Run 对外主函数, 用于运行http(s) server，函数监听ctx.Done()来停止服务器
// 如果ctx为nil, 则自动监听Ctrl+C或者进程结束信号来结束server
func (c *HttpServer) Run(ctx context.Context, configServerFunc func(server *http.Server) error) error {

	server := c.BuildServer()

	go func() {
		select {
		case <-c.listenContext(ctx).Done():
		}

		if err := server.Close(); err != nil {
			c.logger.Fatalf("Server closed: %s", err.Error())
		}
	}()

	// 启动http server
	if !c.IsTLS() {

		if configServerFunc != nil {
			if err := configServerFunc(server); err != nil {
				return err
			}
		}

		c.logger.Infof("Start http server on %s", c.GetHost())

		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				c.logger.Info("http server closed")
			} else {
				return err
			}
		}
	} else { // 启动https server
		//
		if err := SetServerTLSCerts(server, c.GetCertificates()); err != nil {
			return err
		}

		if configServerFunc != nil {
			if err := configServerFunc(server); err != nil {
				return err
			}
		}

		c.logger.Infof("Start https server on %s", c.GetHost())

		if err := server.ListenAndServeTLS("", ""); err != nil {
			if err == http.ErrServerClosed {
				c.logger.Info("https server closed")
			} else {
				return err
			}
		}
	}

	return nil
}

// 监听停止信号, ctx为nil时只收听进程退出信号
func (c *HttpServer) listenContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx1, cancel := context.WithCancel(context.Background())
		termChan := make(chan os.Signal)
		//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
		signal.Notify(termChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			select {
			case <-termChan:
				c.logger.Info("exit signal of process received.")
				cancel()
			}
		}()
		return ctx1
	} else {
		return ctx
	}
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
