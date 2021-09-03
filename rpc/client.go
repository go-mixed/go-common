package rpc

import (
	"go-common/utils"
	"net/rpc"
)

type Client struct {
	network string
	address string
	logger  utils.ILogger
	*rpc.Client
}

func NewClient(network, address string, logger utils.ILogger) (*Client, error) {
	client, err := rpc.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return &Client{
		network,
		address,
		logger,
		client,
	}, nil
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
