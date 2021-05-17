package web

import (
	"crypto/tls"
	"go-common/utils"
	"net/http"
)


type Certificate struct {
	CertFile string
	KeyFile string
}

type IServerConfig interface {
	http.Handler
	GetHost() string
	IsTLS() bool
	GetCertificates() []*Certificate
}

type domainConfig struct {
	cert    *Certificate
	domains []string
	handler http.Handler
}

type ServerConfig struct {
	Host    string
	domains []*domainConfig
}

func NewServerConfig(host string) *ServerConfig {
	return &ServerConfig{
		Host:    host,
		domains: make([]*domainConfig, 0, 1),
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

func (c *ServerConfig) AddDomain(domains []string, handler http.Handler, cert *Certificate) *ServerConfig {
	c.domains = append(c.domains, &domainConfig{
		cert:    cert,
		domains: domains,
		handler: handler,
	})

	return c
}

func (c *ServerConfig) GetHost() string {
	return c.Host
}

func (c *ServerConfig) IsTLS() bool {
	for _, domain := range c.domains {
		if domain.cert != nil {
			return true
		}
	}
	return false
}

func (c *ServerConfig) GetCertificates() []*Certificate {
	var certs []*Certificate
	for _, domain := range c.domains {
		certs = append(certs, domain.cert)
	}
	return certs
}

func (c *ServerConfig) match(domain string) http.Handler {
	if len(c.domains) == 1 {
		return c.domains[0].handler
	}
	return nil
}

func (c *ServerConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	isTLS := c.IsTLS()
	if isTLS {
		for _, domain := range c.domains {
			if domain.cert == nil {
				panic("all domains must have tls certs.")
			}
		}
	}

	if handler := c.match(r.Host); handler != nil {
		handler.ServeHTTP(w, r)
	}
}

// RunServer
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
	if ! serverConfig.IsTLS() {

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