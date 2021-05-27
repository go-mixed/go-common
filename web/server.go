package web

import (
	"crypto/tls"
	"fmt"
	"go-common/cache"
	"go-common/utils"
	"go-common/utils/list"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strings"
	"sync"
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

type HttpServer struct {
	Host                 string
	orderedDomainConfigs []*DomainConfig
	middleware           []Middleware
	// 真正执行的handler入口
	handlerStack         http.Handler
	certs                []*Certificate
	// 为了加快命中, 会将域名所指向的handler进行缓存,此处是该缓存过期市场, 小于60s会修改为60s, 无法设为永久
	domainCacheExpired time.Duration
	mu sync.Mutex
	logger *zap.SugaredLogger
	domainCache *cache.Cache
}

func NewHttpServer(host string) *HttpServer {
	s := &HttpServer{
		Host:                 host,
		orderedDomainConfigs: make([]*DomainConfig, 0, 1),
		domainCacheExpired: 60 * time.Second,
		mu: sync.Mutex{},
		logger: utils.GetSugaredLogger(),
		domainCache: cache.New(60 * time.Second, 30 * time.Second),
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

// SetDomainCacheExpired 设置匹配到域名的缓存过期时间
func (c *HttpServer) SetDomainCacheExpired(domainCacheExpired time.Duration) {
	c.domainCacheExpired = utils.If(domainCacheExpired < 60 * time.Second, 60 * time.Second, domainCacheExpired).(time.Duration)
}


// ContainsDomain 是否包含此域名, 此函数是判断完全相等, 如果需要匹配通配符, 使用 MatchDomain
func (c *HttpServer) ContainsDomain(domain string) bool {
	return list.Find(c.orderedDomainConfigs, func(value interface{}) bool {
		return strings.EqualFold(value.(*DomainConfig).domain, domain)
	}) >= 0
}

// ContainsCert 是否包含此证书, 需要cert/key都相等
func (c *HttpServer) ContainsCert(cert *Certificate) bool {
	return list.Find(c.certs, func(value interface{}) bool {
		_v := value.(*Certificate)
		return os.SameFile(cert.CertFileInfo(), _v.CertFileInfo()) && os.SameFile(cert.KeyFileInfo(), _v.KeyFileInfo())
	}) >= 0
}

// AddServeHandler 添加域名, serveHTTP, 证书
// 可以使用 AddCertificate 传递证书
// 注意: 如果有传递证书，证书DNS Name必须包含所传递的domains（此函数并不检查），不然，需要分多次添加
func (c *HttpServer) AddServeHandler(domains utils.Domains, handler http.Handler, cert *Certificate) error {
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
	utils.SortDomains(&domainConfigs, func(v interface{}) string {
		return v.(*DomainConfig).domain
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
func (c *HttpServer) Use(fn... Middleware) {
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

func (c *HttpServer) SetLogger(logger *zap.SugaredLogger) {
	c.logger = logger
}

func (c *HttpServer) AddCertificate(certs... *Certificate) error  {
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
	c.domainCache.Flush()
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

// MatchDomain 匹配域名, 使用通配符的方式，如果只添加了一个domain，则不论是否匹配都返回该serveHandler
func (c *HttpServer) MatchDomain(domain string) *DomainConfig {
	// 只有一个的情况快速返回
	if len(c.orderedDomainConfigs) == 1 && c.orderedDomainConfigs[0].domain == "*" {
		return c.orderedDomainConfigs[0]
	}

	// 此处可以用Lru cache做持久保存 但是因为域名可能会解绑, 所以需要过期时间
	domainConfig, _ := c.domainCache.Remember(fmt.Sprintf("domain-handlerStack-%s", domain), 60 * time.Second, func() (interface{}, error) {
		c.mu.Lock()
		defer c.mu.Unlock()

		for _, domainConfig := range c.orderedDomainConfigs {
			if utils.WildcardMatch(domainConfig.domain, domain) {
				return domainConfig, nil
			}
		}
		return nil, nil
	})

	if d, ok := domainConfig.(*DomainConfig); ok {
		return d
	}
	return nil
}

func (c *HttpServer) defaultHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if domainConfig := c.MatchDomain(r.Host); domainConfig != nil {
			domainConfig.handler.ServeHTTP(w, r)
		}
	}
}

// ServeHTTP HTTP入口函数，匹配域名之后 分发到不同handler
func (c *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.handlerStack.ServeHTTP(w, r)
}

// Run 对外主函数, 用于运行http(s) server，并且可以监听stopChan来停止服务器
func (c *HttpServer) Run(stopChan <-chan bool) error {

	sugarLogger := utils.GetSugaredLogger()

	server := &http.Server{
		Addr:    c.GetHost(),
		Handler: c,
	}

	go func() {
		<-stopChan

		if err := server.Close(); err != nil {
			sugarLogger.Fatal("Server Close: ", err)
		}
	}()

	// 启动http server
	if !c.IsTLS() {

		sugarLogger.Infof("Start http server on %s", c.GetHost())

		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				sugarLogger.Info("http server closed")
			} else {
				return err
			}
		}
	} else { // 启动https server
		//
		if err := SetServerTLSCerts(server, c.GetCertificates()); err != nil {
			return err
		}

		sugarLogger.Infof("Start https server on %s", c.GetHost())

		if err := server.ListenAndServeTLS("", ""); err != nil {
			if err == http.ErrServerClosed {
				sugarLogger.Info("https server closed")
			} else {
				return err
			}
		}
	}

	return nil
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
