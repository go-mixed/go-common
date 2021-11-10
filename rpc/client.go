package rpc

import (
	"errors"
	"github.com/silenceper/pool"
	"go-common/utils"
	"io"
	"net/rpc"
	"runtime"
	"time"
)

type Client struct {
	network     string
	address     string
	logger      utils.ILogger
	channelPool pool.Pool
}

func NewClient(network, address string, logger utils.ILogger) (*Client, error) {

	client := &Client{
		network,
		address,
		logger,
		nil,
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

type Request []interface{}
type Response []interface{}

func (c *Client) CallArgs(methodName string, args ...interface{}) (Response, error) {
	response := Response{}
	if err := c.Call(methodName, args, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) Call(serviceMethod string, args interface{}, reply interface{}) error {
	client, err := c.GetClient()
	if err != nil {
		return err
	}
	err = client.Call(serviceMethod, args, reply)
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, rpc.ErrShutdown) {
		c.channelPool.Put(client)
	}

	return err
}

func (c *Client) GetClient() (*rpc.Client, error) {
	client, err := c.channelPool.Get()
	if err != nil {
		return nil, err
	}
	return client.(*rpc.Client), nil
}

func (c *Client) Factory() (interface{}, error) {
	return rpc.Dial(c.network, c.address)
}

func (c *Client) PingClient(i interface{}) error {
	return nil
}

func (c *Client) CloseClient(i interface{}) error {
	return i.(*rpc.Client).Close()
}

func (c *Client) Close() error {
	c.channelPool.Release()
	return nil
}
