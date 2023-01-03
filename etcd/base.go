package etcd

import (
	"context"
	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/go-mixed/go-common.v1/cache.v1"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
)

func NewEtcdCache(client *clientv3.Client, logger utils.ILogger) *Etcd {
	c := &Etcd{
		Cache: cache.Cache{
			Ctx:    core.If(client.Ctx() != nil, client.Ctx(), context.Background()),
			Logger: logger,

			EncodeFunc: text_utils.JsonMarshalToBytes,
			DecodeFunc: text_utils.JsonUnmarshalFromBytes,
		},
		EtcdClient: client,
	}
	c.L2Cache = cache.NewL2Cache(c, logger)
	return c
}

func ConnectToEtcd(options *clientv3.Config, logger utils.ILogger) (*Etcd, error) {

	client, err := clientv3.New(*options)
	if err != nil {
		return nil, err
	}

	// testing connect to etcd
	ctx, cancel := context.WithTimeout(options.Context, options.DialTimeout)
	defer cancel()

	status, err := client.Status(ctx, options.Endpoints[0])
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errors.Errorf("dial to etcd endpoint [%s] timeout", options.Endpoints[0])
		}
		return nil, err
	} else if status == nil {
		return nil, errors.Errorf("the status from etcd was nil")
	}

	return NewEtcdCache(client, logger), nil
}
