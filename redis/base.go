package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"gopkg.in/go-mixed/go-common.v1/cache.v1"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
)

// NewRedisCache
// 注意, 此类中 Range/ScanRange/ScanRangeCallback 方法只有后端是pika时才能调用, 不然会panic
// 当后端是pika时, ScanPrefix/ScanPrefixCallback都将会使用pika原生函数来实现
// 当后端是redis时, 尽量避免调用ScanPrefix/ScanPrefixCallback 因为redis在遍历执行scan时非常的慢
func NewRedisCache(client redis.UniversalClient, logger utils.ILogger, isPika bool) *Redis {
	c := &Redis{
		Cache: cache.Cache{
			Ctx:    context.Background(),
			Logger: logger,
		},
		RedisClient: client,
		IsPika:      isPika,
	}
	c.SetEncoderFunc(textUtils.JsonMarshalToBytes)
	c.SetDecoderFunc(textUtils.JsonUnmarshalFromBytes)
	c.L2Cache = cache.NewL2Cache(c, logger)
	return c
}

func ConnectToRedis(options *redis.UniversalOptions, logger utils.ILogger, isPika bool) (*Redis, error) {

	client := redis.NewUniversalClient(options)

	_, err := client.Ping(context.TODO()).Result()

	if err != nil {
		return nil, err
	}

	return NewRedisCache(client, logger, isPika), nil
}
