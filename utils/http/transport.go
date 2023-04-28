package httpUtils

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TransportOptions struct {
	// 连接超时，用于net.Dialer.Timeout，0是永远等待
	ConnectTimeout time.Duration `yaml:"connect_timeout" validate:"min=0"`
	// Expect: 100-continue头的等待上游的超时时间
	ExpectContinueTimeout time.Duration `yaml:"expect_continue_timeout" validate:"min=0"`
	// TLS 握手超时，0是永远等待
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout" validate:"min=0"`

	// 发起keep alive的间隔时间，用于net.Dialer.KeepAlive，0表示自动
	KeepAlive time.Duration `yaml:"keep_alive" validate:"min=0"`
	// true为关闭keep alive
	DisableKeepAlives bool `yaml:"disable_keep_alives"`

	// 连接池配置

	// 空闲超时
	IdleConnTimeout time.Duration `yaml:"idle_conn_timeout" validate:"min=0"`
	// 最大空闲连接数
	MaxIdleConns int `yaml:"max_idle_conns" validate:"min=0"`
	// 每个host最大空闲数，所有hosts的空闲总数由MaxIdleConns控制
	MaxIdleConnsPerHost int `yaml:"max_idle_conns_per_host" validate:"min=0"`
	// 每个host最大连接数，0是不限制
	MaxConnsPerHost int `yaml:"max_conns_per_host" validate:"min=0"`

	// 尝试连接上游时优先使用http2（会自动适配）
	ForceAttemptHTTP2 bool `yaml:"force_attempt_http2"`
	// 关闭http的压缩
	DisableCompression bool `yaml:"disable_compression"`

	// 返回头的超时时间，0是永远等待
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout" validate:"min=0"`
	// 返回头的最大字节，0表示不限制
	MaxResponseHeaderBytes int64 `yaml:"max_response_header_bytes" validate:"min=0"`
	// 写入的缓冲大小，0表示自动
	WriteBufferSize int `yaml:"write_buffer_size" validate:"min=0"`
	// 读取的缓冲大小，0表示自动
	ReadBufferSize int `yaml:"read_buffer_size" validate:"min=0"`

	Hosts map[string]string `yaml:"hosts"`
	Dns   []string          `yaml:"dns"`
}

type Transport struct {
	*http.Transport

	options TransportOptions

	hosts         map[string]string
	sortedDomains Domains
	dns           []string
	httpDialer    *net.Dialer
}

func DefaultTransportOptions() TransportOptions {
	return TransportOptions{
		ConnectTimeout:        30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,

		KeepAlive:         30 * time.Second,
		DisableKeepAlives: false,

		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: http.DefaultMaxIdleConnsPerHost,
		MaxConnsPerHost:     0,

		ForceAttemptHTTP2:  true,
		DisableCompression: false,

		ResponseHeaderTimeout:  30 * time.Second,
		MaxResponseHeaderBytes: 0,
		WriteBufferSize:        0,
		ReadBufferSize:         0,

		Hosts: map[string]string{},
		Dns:   []string{},
	}
}

func NewHttpTransport(options TransportOptions) *Transport {
	t := &Transport{
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
		},
		hosts:         map[string]string{},
		sortedDomains: Domains{},
	}

	t.DialContext = t.dialContext
	t.SetOptions(options)

	t.httpDialer = &net.Dialer{
		Timeout:   t.options.ConnectTimeout,
		KeepAlive: t.options.KeepAlive,
		Resolver:  t.defaultResolver(),
	}

	return t
}

// SetOptions 重设Transport的配置
//
//	注意：options中hosts和dns的值如果不为nil，会覆盖原有的设置
func (t *Transport) SetOptions(options TransportOptions) *Transport {
	t.options = options
	t.ForceAttemptHTTP2 = options.ForceAttemptHTTP2
	t.MaxIdleConns = options.MaxIdleConns
	t.MaxConnsPerHost = options.MaxConnsPerHost
	t.IdleConnTimeout = options.IdleConnTimeout
	t.MaxIdleConnsPerHost = options.MaxIdleConnsPerHost
	t.TLSHandshakeTimeout = options.TLSHandshakeTimeout
	t.ExpectContinueTimeout = options.ExpectContinueTimeout
	t.DisableKeepAlives = options.DisableKeepAlives
	t.DisableCompression = options.DisableCompression
	t.ResponseHeaderTimeout = options.ResponseHeaderTimeout
	t.MaxResponseHeaderBytes = options.MaxResponseHeaderBytes
	t.WriteBufferSize = options.WriteBufferSize
	t.ReadBufferSize = options.ReadBufferSize
	if options.Hosts != nil {
		t.SetHosts(options.Hosts)
	}
	if options.Dns != nil {
		t.SetDns(options.Dns)
	}
	return t
}

// 用于http服务的Transport.DialContext
func (t *Transport) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, _ := net.SplitHostPort(addr)

	// 从hosts中查找domain => ip，泛域名的优先级最低
	// 注意：如果设置了泛域名
	if ok, domain := t.sortedDomains.Match(host); ok {
		ip, _ := t.hosts[domain]
		if ip != "" {
			addr = net.JoinHostPort(ip, port)
		}
	}
	return t.httpDialer.DialContext(ctx, network, addr)
}

// DisableTlsVerify disable tls certificate verify
func (t *Transport) DisableTlsVerify() *Transport {
	t.TLSClientConfig.InsecureSkipVerify = true
	return t
}

// EnableTlsVerify enable tls certificate verify
func (t *Transport) EnableTlsVerify() *Transport {
	t.TLSClientConfig.InsecureSkipVerify = false
	return t
}

// SetResolver set a resolver to transport, dns will be ignored if resolver is set
//
//	如果自定义了resolver，就会忽略DNS的配置
func (t *Transport) SetResolver(resolver *net.Resolver) *Transport {
	t.httpDialer.Resolver = resolver
	return t
}

// SetHosts Set hosts to resolver, support wildcard domain like *.example.com
// [domain => ip, ...]
//
//   - domains will be converted to lowercase
//   - wildcard domain always has the lowest priority
func (t *Transport) SetHosts(hosts map[string]string) *Transport {
	t.hosts = map[string]string{}
	t.sortedDomains = Domains{}

	for domain, ip := range hosts {
		domain = strings.ToLower(domain)

		t.hosts[domain] = ip
		t.sortedDomains = append(t.sortedDomains, domain)
	}
	t.sortedDomains.Sort()
	return t
}

// AddHostsLine Add a line of hosts, support multiple domains and wildcard domain like *.example.com
//
//   - domains will be converted to lowercase
//   - wildcard domain always has the lowest priority
func (t *Transport) AddHostsLine(ip string, domains ...string) *Transport {
	for _, domain := range domains {
		domain = strings.ToLower(domain)

		t.hosts[domain] = ip
		t.sortedDomains = append(t.sortedDomains, domain)
	}
	t.sortedDomains.Sort()
	return t
}

// SetDns set dns to resolver，只能输入IPV4，会使用UDP去查询
func (t *Transport) SetDns(dns []string) *Transport {
	t.dns = dns
	return t
}

// SetProxy set a http proxy to transport，
//
//	eg: http://127.0.0.1:1080、socks5://127.0.0.1:1080
func (t *Transport) SetProxy(proxyUrl string) *Transport {
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		return url.Parse(proxyUrl)
	}
	return t
}

// build or get resolver，自定义了dns则使用UDP查询DNS，否则使用系统的DNS方式
func (t *Transport) defaultResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		// DNS 的Dialer，如果自定过了dns的ip，就使用UDP的53端口通讯。
		// 如果需要TCP通讯，就需要自定义 SetResolver
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout:   t.options.ConnectTimeout, // 此处使用全局的ConnectTimeout和KeepAlive
				KeepAlive: t.options.KeepAlive,
			}

			if len(t.dns) > 0 { // 使用自定义的DNS服务器
				return d.DialContext(ctx, "udp", t.dns[0])
			}
			return d.DialContext(ctx, network, address)
		},
	}
}

func (t *Transport) MakeClient() *http.Client {
	return &http.Client{
		Transport: t,
	}
}
