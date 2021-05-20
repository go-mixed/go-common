package web

import (
	"crypto/tls"
	"fmt"
	"go-common/utils"
	"go-common/utils/list"
	"net/http"
	"os"
	"strings"
)

type Certificate struct {
	CertFile string `json:"cert"`
	KeyFile  string `json:"key"`

	certFileInfo os.FileInfo
	keyFileInfo  os.FileInfo
}

type IServerConfig interface {
	http.Handler
	GetHost() string
	IsTLS() bool
	GetCertificates() []*Certificate
}

type domainConfig struct {
	domain  string
	handler http.Handler
}

type ServerConfig struct {
	Host                 string
	orderedDomainConfigs []*domainConfig
	certs                []*Certificate
	running              bool
}

func NewServerConfig(host string) *ServerConfig {
	return &ServerConfig{
		Host:                 host,
		orderedDomainConfigs: make([]*domainConfig, 0, 1),
	}
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

func (c *ServerConfig) HasDefaultDomain() bool {
	return c.ContainsDomain("*")
}

// ContainsDomain 是否包含此域名, 此函数是判断完全相等, 如果需要匹配通配符, 使用 MatchDomain
func (c *ServerConfig) ContainsDomain(domain string) bool {
	return list.Find(c.orderedDomainConfigs, func(value interface{}) bool {
		return strings.EqualFold(value.(*domainConfig).domain, domain)
	}) >= 0
}

// ContainsCert 是否包含此证书, 需要cert/key都相等
func (c *ServerConfig) ContainsCert(cert *Certificate) bool {
	return list.Find(c.certs, func(value interface{}) bool {
		_v := value.(*Certificate)
		return os.SameFile(cert.CertFileInfo(), _v.CertFileInfo()) && os.SameFile(cert.KeyFileInfo(), _v.KeyFileInfo())
	}) >= 0
}

// AddServeHandler 添加域名, serveHTTP, 证书
// 注意: 如果有传递证书，证书DNS Name必须包含所传递的domains（此函数并不检查），不然，需要分多次添加
func (c *ServerConfig) AddServeHandler(domains utils.Domains, handler http.Handler, certs []*Certificate) error {
	if c.running {
		return fmt.Errorf("server is running, can not add handler anymore")
	}

	for _, cert := range certs {
		if cert.CertFileInfo() == nil || cert.KeyFileInfo() == nil {
			return fmt.Errorf("cert \"%s\" or key \"%s\" is invalid", cert.CertFile, cert.KeyFile)
		}
		if !c.ContainsCert(cert) { // 添加cert列表
			c.certs = append(c.certs, cert)
		}
	}

	// 添加
	domainConfigs := c.orderedDomainConfigs[:]
	for _, domain := range domains {
		domain = strings.ToLower(domain)
		if c.ContainsDomain(domain) {
			return fmt.Errorf("domain duplicate: \"%s\" is already in the server config", domain)
		}

		domainConfigs = append(domainConfigs, &domainConfig{
			domain:  domain,
			handler: handler,
		})
	}

	// 按照域名的特有方式进行排序
	utils.SortDomains(&domainConfigs, func(v interface{}) string {
		return v.(*domainConfig).domain
	})

	c.orderedDomainConfigs = domainConfigs
	return nil
}

// SetDefaultServeHandler 设置默认的ServeHandler, 即添加一个通配符*的域名
func (c *ServerConfig) SetDefaultServeHandler(handler http.Handler, certs []*Certificate) error {
	return c.AddServeHandler([]string{"*"}, handler, certs)
}

// GetHost 获取待监听的HOST, 比如: 0.0.0.0:80
func (c *ServerConfig) GetHost() string {
	return c.Host
}

// IsTLS 是否提供TLS服务
func (c *ServerConfig) IsTLS() bool {
	return len(c.certs) > 0
}

// GetCertificates 获取证书列表
func (c *ServerConfig) GetCertificates() []*Certificate {
	return c.certs
}

// MatchDomain 匹配域名, 使用通配符的方式，如果只添加了一个domain，则不论是否匹配都返回该serveHandler
func (c *ServerConfig) MatchDomain(domain string) http.Handler {
	if len(c.orderedDomainConfigs) == 1 {
		return c.orderedDomainConfigs[0].handler
	}
	for _, domainConfig := range c.orderedDomainConfigs {
		if utils.WildcardMatch(domainConfig.domain, domain) {
			return domainConfig.handler
		}
	}
	return nil
}

// ServeHTTP HTTP入口函数，匹配域名之后 分发到不同handler
func (c *ServerConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//isTLS := c.IsTLS()
	//if isTLS {
	//	for _, domain := range c.orderedDomainConfigs {
	//		if domain.cert == nil {
	//			panic("some domains have no tls certs.")
	//		}
	//	}
	//}

	if handler := c.MatchDomain(r.Host); handler != nil {
		handler.ServeHTTP(w, r)
	}
}

// RunServer 对外主函数, 用于运行http(s) server，并且可以监听stopChan来停止服务器
func RunServer(stopChan <-chan bool, serverConfig IServerConfig) error {

	sugarLogger := utils.GetSugaredLogger()

	server := &http.Server{
		Addr:    serverConfig.GetHost(),
		Handler: serverConfig,
	}

	go func() {
		<-stopChan

		if err := server.Close(); err != nil {
			sugarLogger.Fatal("Server Close: ", err)
		}
	}()

	// 启动http server
	if !serverConfig.IsTLS() {

		sugarLogger.Infof("Start http server on %s", serverConfig.GetHost())

		if err := server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				sugarLogger.Info("http server closed")
			} else {
				return err
			}
		}
	} else { // 启动https server
		//
		if err := SetServerTLSCerts(server, serverConfig.GetCertificates()); err != nil {
			return err
		}

		sugarLogger.Infof("Start https server on %s", serverConfig.GetHost())

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
