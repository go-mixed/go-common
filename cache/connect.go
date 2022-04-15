package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"go-common/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func ConnectToRedis(options *redis.UniversalOptions, logger utils.ILogger, isPika bool) (*Redis, error) {

	client := redis.NewUniversalClient(options)

	_, err := client.Ping(context.TODO()).Result()

	if err != nil {
		return nil, err
	}

	return NewRedisCache(client, logger, isPika), nil
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
			return nil, fmt.Errorf("dial to etcd endpoint [%s] timeout", options.Endpoints[0])
		}
		return nil, err
	} else if status == nil {
		return nil, fmt.Errorf("the status from etcd was nil")
	}

	return NewEtcdCache(client, logger), nil
}
