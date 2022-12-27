package cache

import (
	"context"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"go-common/utils"
	"go-common/utils/conv"
	"go-common/utils/core"
	"go-common/utils/text"
	"strings"
	"time"
)

type Redis struct {
	Cache
	IsPika      bool
	RedisClient redis.UniversalClient
}

func (c *Redis) WithContext(ctx context.Context) *Redis {
	newRedis := *c
	newRedis.Ctx = ctx
	return &newRedis
}

func (c *Redis) SetNoExpiration(key string, val any) error {
	return c.Set(key, val, 0)
}

func (c *Redis) Exists(key string) bool {
	n, err := c.RedisClient.Exists(c.Ctx, key).Result()
	if err == redis.Nil { // 无此数据
		c.Logger.Debugf("[Redis]key not exists: %s", key)
		return false
	} else if err != nil {
		c.Logger.Debugf("[Redis]error of key %s", key, err.Error())
		return false
	}
	return n > 0
}

func (c *Redis) Incr(key string) int64 {
	n, err := c.RedisClient.Incr(c.Ctx, key).Result()
	if err != nil {
		c.Logger.Debugf("[Redis]error of key %s", key, err.Error())
		return 0
	}
	return n
}

func (c *Redis) Decr(key string) int64 {
	n, err := c.RedisClient.Decr(c.Ctx, key).Result()
	if err != nil {
		c.Logger.Debugf("[Redis]error of key %s", key, err.Error())
		return 0
	}
	return n
}

func (c *Redis) Del(key string) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]Del %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.RedisClient.Del(c.Ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *Redis) Set(key string, val any, expiration time.Duration) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.RedisClient.Set(c.Ctx, key, text_utils.ToString(val, true), expiration).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *Redis) Get(key string, result any) ([]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]Get %s, %0.6f", key, time.Since(now).Seconds())
	}()
	val, err := c.RedisClient.Get(c.Ctx, key).Result()
	if err == redis.Nil { // 无此数据
		c.Logger.Debugf("[Redis]key not exists: %s", key)
		return nil, nil
	} else if err != nil {
		c.Logger.Debugf("[Redis]error of key %s", key, err.Error())
		return nil, err
	} else if val == "" { // 返回为空数据也不正确
		c.Logger.Debugf("[Redis]empty value of key %s", key)
		return nil, nil
	}

	if !core.IsInterfaceNil(result) {
		if err := text_utils.JsonUnmarshal(val, result); err != nil {
			c.Logger.Errorf("[Redis]json unmarshal: %s of error: %s", val, err.Error())
			return []byte(val), err
		}
	}
	return []byte(val), nil
}

func (c *Redis) MGet(keys []string, result any) (utils.KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]MGet %v, %0.6f", keys, time.Since(now).Seconds())
	}()
	val, err := c.RedisClient.MGet(c.Ctx, keys...).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(val) == 0 { // 一个数据都没找到
		return nil, nil
	}

	kvs := utils.KVs{}
	for i, v := range val {
		if _v, ok := v.(string); ok {
			kvs = kvs.Append(keys[i], []byte(_v))
		} else {
			kvs = kvs.Append(keys[i], nil)
		}
	}
	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), result); err != nil {
			c.Logger.Errorf("[Redis]json unmarshal: %v of error: %s", kvs.Values(), err.Error())
			return nil, err
		}
	}
	return kvs, nil
}

func (c *Redis) Keys(keyPrefix string) ([]string, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]Keys %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"
	val, err := c.RedisClient.Keys(c.Ctx, keyPrefix).Result()
	if err == redis.Nil { // 无此数据
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return val, nil
}

func (c *Redis) ScanPrefix(keyPrefix string, result any) (utils.KVs, error) {
	if c.IsPika {
		return c.pikaScanPrefix(keyPrefix, result)
	}
	// 以下是redis中的实现
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanPrefix %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"

	var cursor uint64
	var err error
	replicateKeys := map[string]bool{}
	var keys []string

	for {
		var _keys []string
		_keys, cursor, err = c.Scan(keyPrefix, cursor, 10)
		if err != nil {
			return nil, err
		}
		if cursor == 0 {
			break
		}

		// redis的scan会有重复的key出现, 在此处去重
		for _, key := range _keys {
			if _, ok := replicateKeys[key]; ok {
				continue
			}
			keys = append(keys, key)
			replicateKeys[key] = true
		}
	}

	return c.MGet(keys, result)
}

func (c *Redis) ScanPrefixCallback(keyPrefix string, callback func(kv *utils.KV) error) (int64, error) {
	if c.IsPika {
		return c.pikaScanPrefixCallback(keyPrefix, callback)
	}
	// 以下是redis中的实现
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"

	var cursor uint64
	var err error
	var read int64
	replicateKeys := map[string]bool{}
	for {
		var keys []string
		var _keys []string
		_keys, cursor, err = c.Scan(keyPrefix, cursor, 10)
		if err != nil {
			return read, err
		}
		if cursor == 0 {
			return read, nil
		}

		// redis的scan会有重复的key出现, 在此处去重
		for _, key := range _keys {
			if _, ok := replicateKeys[key]; ok {
				continue
			}
			keys = append(keys, key)
			replicateKeys[key] = true
		}

		if len(keys) > 0 {
			kvs, err := c.MGet(keys, nil)
			if err != nil {
				return read, err
			}
			for _, kv := range kvs {
				read++

				if err := callback(kv); err != nil {
					return read, err
				}
			}
		}
	}
}

// Scan redis原生函数(pika也支持), 根据的keyPattern表达式, 以及游标和页码 返回所有匹配的keys
// 注意 redis在遍历scan时非常慢
func (c *Redis) Scan(keyPattern string, cursor uint64, count int64) ([]string, uint64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]scan %s, cursor %d, count %d, %0.6f", keyPattern, cursor, count, time.Since(now).Seconds())
	}()

	var err error
	var keys []string
	keys, cursor, err = c.RedisClient.Scan(c.Ctx, cursor, keyPattern, count).Result()
	if err == redis.Nil { // 无此数据
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	return keys, cursor, err
}

func (c *Redis) Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	if !c.IsPika {
		panic("only use this method in pika")
	}
	params := []any{
		"pkscanrange", "string_with_value", keyStart, keyEnd,
	}

	if keyPrefix != "" {
		keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"
		params = append(params, "MATCH", keyPrefix)
	}
	if limit > 0 {
		params = append(params, "LIMIT", conv.I64toa(limit))
	}

	_res, err := c.RedisClient.Do(c.Ctx, params...).Result()
	if err == redis.Nil {
		return "", nil, nil
	} else if err != nil {
		return "", nil, err
	}

	res, ok := _res.([]any)
	if !ok || len(res) <= 1 {
		return "", nil, nil
	}
	nextKey, ok := res[0].(string)
	if !ok {
		return "", nil, errors.Errorf("scan range returns an invalid next-key")
	}
	_kv, ok := res[1].([]any)
	if !ok {
		return "", nil, errors.Errorf("scan range returns an invalid k/v")
	}
	if len(_kv) <= 0 {
		return nextKey, nil, nil
	}
	// build the kvs
	kvs := utils.KVs{}
	for i := 0; i < len(_kv); i += 2 {
		k := _kv[i].(string)
		v := _kv[i+1].(string)
		if v == "" {
			kvs = kvs.Append(k, nil)
		} else {
			kvs = kvs.Append(k, []byte(v))
		}
	}
	return nextKey, kvs, nil
}

func (c *Redis) pikaScanPrefix(keyPrefix string, result any) (utils.KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanPrefix %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	return c.scanPrefix(keyPrefix, result, c.Range)
}

func (c *Redis) pikaScanPrefixCallback(keyPrefix string, callback func(kv *utils.KV) error) (int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	return c.scanPrefixCallback(keyPrefix, callback, c.Range)
}

func (c *Redis) ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, result any) (string, utils.KVs, error) {
	if !c.IsPika {
		panic("only use this method in pika")
	}
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanRange: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()

	return c.scanRange(keyStart, keyEnd, keyPrefix, limit, result, c.Range)
}

func (c *Redis) ScanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error) (string, int64, error) {
	if !c.IsPika {
		panic("only use this method in pika")
	}
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[Redis]ScanRangeCallback: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()

	return c.scanRangeCallback(keyStart, keyEnd, keyPrefix, limit, callback, c.Range)
}

func (c *Redis) Close() error {
	return c.RedisClient.Close()
}
