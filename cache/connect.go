package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go-common/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func ConnectRedis(options *redis.UniversalOptions, logger utils.ILogger, isPika bool) (*Redis, error) {

	client := redis.NewUniversalClient(options)

	_, err := client.Ping(context.TODO()).Result()

	if err != nil {
		return nil, err
	}

	return NewRedisCache(client, logger, isPika), nil
}

func ConnectEtcd(options *clientv3.Config, logger utils.ILogger) (*Etcd, error) {

	client, err := clientv3.New(*options)
	if err != nil {
		return nil, err
	}

	return NewEtcdCache(client, logger), nil
}
