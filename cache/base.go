package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go-common/utils"
	"go-common/utils/core"
	"go-common/utils/text"
	"go.etcd.io/etcd/client/v3"
	"time"
)

type ICache interface {
	// L2 得到本Cache的二级缓存对象
	L2() IL2Cache
	// Get 查询key的值, 并尝试将其JSON值导出到actual 如果无需导出, actual 传入nil
	Get(key string, actual any) ([]byte, error)
	// MGet 查询多个keys, 返回所有符合要求K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 例子:
	// var result []User
	// RedisGet(keys, &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	MGet(keys []string, actual any) (utils.KVs, error)

	// Keys keyPrefix为前缀 返回所有符合要求的keys
	// 注意: 遇到有太多的匹配性, 会阻塞cache的运行
	Keys(keyPrefix string) ([]string, error)
	// Range 在 keyStart~keyEnd中查找符合keyPrefix要求的KV, limit 为 0 表示不限数量
	// 返回nextKey, kv列表, 错误
	Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error)
	// ScanPrefix keyPrefix为前缀, 返回所有符合条件的K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 注意: 不要在keyPrefix中或结尾加入*
	// 例子:
	// var result []User
	// ScanPrefix("users/id/", &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	// 注意: 如果有太多的匹配项, 会阻塞cache的运行. 对于大的量级, 尽量使用 ScanPrefixCallback
	ScanPrefix(keyPrefix string, actual any) (utils.KVs, error)
	// ScanPrefixCallback 根据keyPrefix为前缀 查询出所有K/V 遍历调用callback
	// 如果callback返回nil, 会一直搜索直到再无匹配数据; 如果返回错误, 则立即停止搜索
	// 注意: 即使cache中有大量的匹配项, 也不会被阻塞
	ScanPrefixCallback(keyPrefix string, callback func(kv *utils.KV) error) (int64, error)

	// ScanRange 根据keyStart/keyEnd返回所有符合条件的K/V, 并尝试将JSON数据导出到actual 如果无需导出, actual 传入nil
	// 注意: 返回的结果会包含keyStart/keyEnd
	// 如果keyPrefix不为空, 则在keyStart/keyEnd中筛选出符keyPrefix条件的项目
	// 如果limit = 0 表示不限数量
	// 例子:
	// var result []User
	// 从 "users/id/100" 开始, 取前缀为"users/id/"的100个数据
	// ScanRange("users/id/100", "", "users/id/", 100, &result)
	// 比如取a~z的所有数据, 会包含 "a", "a1", "a2xxxxxx", "yyyyyy", "z"
	// ScanRange("a", "z", "", 0, &result)
	// 注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, actual any) (string, utils.KVs, error)
	// ScanRangeCallback 根据keyStart/keyEnd返回所有符合条件的K/V, 并遍历调用callback
	// 参数定义参见 ScanRange
	// 如果callback返回nil, 会一直搜索直到再无匹配数据; 如果返回错误, 则立即停止搜索
	ScanRangeCallback(keyStart, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error) (string, int64, error)

	// Set 写入KV
	Set(key string, val any, expiration time.Duration) error
	SetNoExpiration(key string, val any) error
	Del(key string) error
}

type Cache struct {
	Ctx     context.Context
	Logger  utils.ILogger
	l2Cache *L2Cache
}

// NewRedisCache
// 注意, 此类中 Range/ScanRange/ScanRangeCallback 方法只有后端是pika时才能调用, 不然会panic
// 当后端是pika时, ScanPrefix/ScanPrefixCallback都将会使用pika原生函数来实现
// 当后端是redis时, 尽量避免调用ScanPrefix/ScanPrefixCallback 因为redis在遍历执行scan时非常的慢
func NewRedisCache(client redis.UniversalClient, logger utils.ILogger, isPika bool) *Redis {
	cache := &Redis{
		Cache: Cache{
			Ctx:    context.Background(),
			Logger: logger,
		},
		RedisClient: client,
		IsPika:      isPika,
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
		EtcdClient: client,
	}
	cache.l2Cache = NewL2Cache(cache, logger)
	return cache
}

func (c *Cache) L2() IL2Cache {
	return c.l2Cache
}

type RangeFunc func(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error)

func (c *Cache) scanRange(keyStart, keyEnd string, keyPrefix string, limit int64, result any, rangeFunc RangeFunc) (string, utils.KVs, error) {
	nextKey, kvs, err := rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, kvs, err
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), result); err != nil {
			return "", nil, err
		}
	}

	return nextKey, kvs, nil
}

func (c *Cache) scanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (string, int64, error) {
	nextKey, kvs, err := rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, 0, err
	}

	var read int64 = 0
	for _, kv := range kvs {
		if err != nil { // 出错的下一个key
			return kv.Key, read, err
		}
		read++

		if err = callback(kv); err != nil {
			//遇到错误时, continue, 根据上面err的判断, 方法会return错误后下一个key, 也就是nextKey
			continue
		}
	}

	return nextKey, read, err
}

func (c *Cache) scanPrefix(keyPrefix string, result any, rangeFunc RangeFunc) (utils.KVs, error) {
	kvs := utils.KVs{}

	var keyStart = keyPrefix
	var keyEnd = clientv3.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs utils.KVs
	for {
		nextKey, _kvs, err = rangeFunc(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return nil, err
		}
		kvs = kvs.Add(_kvs)
		if nextKey == "" {
			break
		}
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), result); err != nil {
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Cache) scanPrefixCallback(keyPrefix string, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (int64, error) {
	var keyStart = keyPrefix
	var keyEnd = clientv3.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs utils.KVs
	var read int64
	for {
		nextKey, _kvs, err = rangeFunc(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return read, err
		}

		for _, kv := range _kvs {
			read++
			if err := callback(kv); err != nil {
				return read, err
			}
		}

		if nextKey == "" {
			return read, nil
		}
	}
}
