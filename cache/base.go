package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go-common/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type ICache interface {
	// L2 得到本Cache的二级缓存对象
	L2() IL2Cache
	// Get 查询key的值, 并尝试将其JSON值导出到actual 如果无需导出, actual 传入nil
	Get(key string, actual interface{}) ([]byte, error)
	// MGet 查询多个keys, 返回所有符合要求K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 例子:
	// var result []User
	// RedisGet(keys, &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	MGet(keys []string, actual interface{}) (KVs, error)

	// Keys keyPrefix为前缀 返回所有符合要求的keys
	// 注意: 遇到有太多的匹配性, 会阻塞整个redis的运行
	Keys(keyPrefix string) ([]string, error)

	// ScanPrefix keyPrefix为前缀, 返回所有符合条件的K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 注意: 不要在keyPrefix中或结尾加入*
	// 例子:
	// var result []User
	// ScanPrefix("users/id/", &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	// 注意: redis中如果有太多的匹配项, 会阻塞整个redis的运行. 对于太大的量级, 尽量使用 ScanPrefixCallback
	ScanPrefix(keyPrefix string, actual interface{}) (KVs, error)
	// ScanPrefixCallback 根据keyPrefix为前缀 查询出所有K/V 遍历调用callback
	// 如果callback返回nil, 会一直搜索直到再无匹配数据; 如果返回错误, 则立即停止搜索
	// 注意: 即使redis中有大量的匹配项, 也不会被阻塞
	ScanPrefixCallback(keyPrefix string, callback func(kv *KV) error) (int64, error)

	// ScanRange 根据keyStart/keyEnd返回所有符合条件的K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 注意: 返回的结果会包含keyStart/keyEnd
	// 如果keyPrefix不为空, 则在keyStart/keyEnd中筛选出符keyPrefix条件的项目
	// 如果limit = 0 表示无限个
	// 例子:
	// var result []User
	// 从 "users/id/100" 开始, 取前缀为"users/id/"的100个数据
	// ScanRange("users/id/100", "", "users/id/", 100, &result)
	// 取a~z的所有数据, 包含 "a", "a1", "a2xxxxxx", "yyyyyy", "z"
	// ScanRange("a", "z", "", 0, &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	// redis不支持 ScanRange, 可使用pika等兼容redis的后端来代替
	ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, actual interface{}) (string, KVs, error)
	// ScanRangeCallback 根据keyStart/keyEnd返回所有符合条件的K/V, 并遍历调用callback
	// 参数定义参见 ScanRange
	// 如果callback返回nil, 会一直搜索直到再无匹配数据; 如果返回错误, 则立即停止搜索
	ScanRangeCallback(keyStart, keyEnd string, keyPrefix string, limit int64, callback func(kv *KV) error) (string, int64, error)

	// Set 写入KV
	Set(key string, val interface{}, expiration time.Duration) error
	SetNoExpiration(key string, val interface{}) error
	Del(key string) error
}

type Cache struct {
	Ctx     context.Context
	Logger  utils.ILogger
	l2Cache *L2Cache
}

type KV struct {
	Key   string
	Value []byte
}

type KVs []*KV

func (s KVs) Append(k string, v []byte) KVs {
	return append(s, &KV{
		Key:   k,
		Value: v,
	})
}

func (s KVs) Add(_new KVs) KVs {
	return append(s, _new...)
}

func (s KVs) Keys() []string {
	var keys []string
	for _, kv := range s {
		keys = append(keys, kv.Key)
	}
	return keys
}

func (s KVs) Values() [][]byte {
	var values [][]byte
	for _, kv := range s {
		values = append(values, kv.Value)
	}
	return values
}

func NewRedisCache(client redis.UniversalClient, logger utils.ILogger) *Redis {
	cache := &Redis{
		Cache: Cache{
			Ctx:    context.Background(),
			Logger: logger,
		},
		redisClient: client,
	}
	cache.l2Cache = NewL2Cache(cache, logger)
	return cache
}

func NewEtcdCache(client *clientv3.Client, logger utils.ILogger) *Etcd {
	cache := &Etcd{
		Cache: Cache{
			Ctx:    context.Background(),
			Logger: logger,
		},
		client: client,
	}
	cache.l2Cache = NewL2Cache(cache, logger)
	return cache
}

func (c *Cache) L2() IL2Cache {
	return c.l2Cache
}
