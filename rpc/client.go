package rpc

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/silenceper/pool"
	"go-common/utils"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"runtime"
	"time"
)

type Client struct {
	network     string
	address     string
	logger      utils.ILogger
	channelPool pool.Pool
	timeout     time.Duration
}

var connected = "200 Connected to Go RPC"

func NewClient(network, address string, logger utils.ILogger) (*Client, error) {

	client := &Client{
		network,
		address,
		logger,
		nil,
		5 * time.Second,
	}

	channelPool, err := pool.NewChannelPool(&pool.Config{
		InitialCap:  runtime.NumCPU(),
		MaxCap:      runtime.NumCPU() * 2,
		MaxIdle:     runtime.NumCPU(),
		Factory:     client.Factory,
		Close:       client.CloseClient,
		Ping:        client.PingClient,
		IdleTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	client.channelPool = channelPool

	return client, nil
}

func NewHttpClient(address string, logger utils.ILogger) (*Client, error) {
	return NewClient("http", address, logger)
}

type Request []any
type Response []any

func (c *Client) CallArgs(methodName string, args ...any) (Response, error) {
	response := Response{}
	if err := c.Call(methodName, args, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) Call(serviceMethod string, args any, reply any) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}

	if c.timeout > 0 {
		// 使用NewTimer并Stop避免time.After内存泄露问题
		after := time.NewTimer(c.timeout)
		defer after.Stop()

		select {
		case <-after.C: // 无需经过下面的错误判断, 直接退出
			return errors.Errorf("rpc client call timeout > %.4fs", c.timeout.Seconds())
		case call := <-client.Go(serviceMethod, args, reply, make(chan *rpc.Call, 1)).Done:
			err = call.Error
		}
	} else {
		err = c.Call(serviceMethod, args, reply)
	}

	if err == nil || (!errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, rpc.ErrShutdown)) {
		c.channelPool.Put(client)
	}

	return err
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *Client) GetClient() (*rpc.Client, error) {
	client, err := c.channelPool.Get()
	if err != nil {
		return nil, err
	}
	return client.(*rpc.Client), nil
}

func (c *Client) Factory() (any, error) {
	if c.network == "http" {
		return c.dialHTTPPath("tcp", c.address, c.timeout, rpc.DefaultRPCPath)
	}

	return c.dial(c.network, c.address, c.timeout)
}

// Dial connects to an RPC server at the specified network address.
func (c *Client) dial(network, address string, timeout time.Duration) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(conn), nil
}

// DialHTTPPath connects to an HTTP RPC server
// at the specified network address and path.
func (c *Client) dialHTTPPath(network, address string, timeout time.Duration, path string) (*rpc.Client, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")

	// Require successful HTTP response
	// before switching to RPC protocol.
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return rpc.NewClient(conn), nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	conn.Close()
	return nil, &net.OpError{
		Op:   "dial-http",
		Net:  network + " " + address,
		Addr: nil,
		Err:  err,
	}
}

func (c *Client) PingClient(i any) error {
	return nil
}

func (c *Client) CloseClient(i any) error {
	return i.(*rpc.Client).Close()
}

func (c *Client) Close() error {
	c.channelPool.Release()
	return nil
}
