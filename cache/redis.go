package cache

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"go-common/utils/conv"
	"go-common/utils/core"
	text_utils "go-common/utils/text"
	"strings"
	"time"
)

type Redis struct {
	Cache
	redisClient redis.UniversalClient
}

func (c *Redis) SetNoExpiration(key string, val interface{}) error {
	return c.Set(key, val, 0)
}

func (c *Redis) Del(key string) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis Del %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.redisClient.Del(c.Ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *Redis) Set(key string, val interface{}, expiration time.Duration) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.redisClient.Set(c.Ctx, key, text_utils.ToString(val, true), expiration).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *Redis) Get(key string, result interface{}) ([]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis Get %s, %0.6f", key, time.Since(now).Seconds())
	}()
	val, err := c.redisClient.Get(c.Ctx, key).Result()
	if err == redis.Nil { // 无此数据
		c.Logger.Debugf("redis key not exists: %s", key)
		return nil, nil
	} else if err != nil {
		c.Logger.Debugf("redis error of key %s", key, err.Error())
		return nil, err
	} else if val == "" { // 返回为空数据也不正确
		c.Logger.Debugf("redis empty value of key %s", key)
		return nil, nil
	}

	if !core.IsInterfaceNil(result) {
		if err := text_utils.JsonUnmarshal(val, result); err != nil {
			c.Logger.Errorf("redis json unmarshal: %s of error: %s", val, err.Error())
			return []byte(val), err
		}
	}
	return []byte(val), nil
}

func (c *Redis) MGet(keys []string, result interface{}) (KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis MGet %v, %0.6f", keys, time.Since(now).Seconds())
	}()
	val, err := c.redisClient.MGet(c.Ctx, keys...).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(val) == 0 { // 一个数据都没找到
		return nil, nil
	}

	kvs := KVs{}
	for i, v := range val {
		if _v, ok := v.(string); ok {
			kvs = kvs.Append(keys[i], []byte(_v))
		} else {
			kvs = kvs.Append(keys[i], nil)
		}
	}
	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), result); err != nil {
			c.Logger.Errorf("redis json unmarshal: %v of error: %s", kvs.Values(), err.Error())
			return nil, err
		}
	}
	return kvs, nil
}

func (c *Redis) Keys(keyPrefix string) ([]string, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis Keys %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"
	val, err := c.redisClient.Keys(c.Ctx, keyPrefix).Result()
	if err == redis.Nil { // 无此数据
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return val, nil
}

func (c *Redis) ScanPrefix(keyPrefix string, result interface{}) (KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis ScanPrefix %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	var cursor uint64
	var err error
	replicateKeys := map[string]bool{}
	var keys []string

	for {
		var _keys []string
		_keys, cursor, err = c.scan(keyPrefix, cursor, 10)
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

func (c *Redis) ScanPrefixCallback(keyPrefix string, callback func(kv *KV) error) (int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	var cursor uint64
	var err error
	var read int64
	replicateKeys := map[string]bool{}
	for {
		var keys []string
		var _keys []string
		_keys, cursor, err = c.scan(keyPrefix, cursor, 10)
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

// Scan 按照的key表达式, 以及游标和页码 返回所有匹配的keys
func (c *Redis) scan(keyPattern string, cursor uint64, count int64) ([]string, uint64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis scan %s, cursor %d, count %d, %0.6f", keyPattern, cursor, count, time.Since(now).Seconds())
	}()
	keyPattern = strings.TrimRight(keyPattern, "*") + "*"
	var err error
	var keys []string
	keys, cursor, err = c.redisClient.Scan(c.Ctx, cursor, keyPattern, count).Result()
	if err == redis.Nil { // 无此数据
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	return keys, cursor, err
}

func (c *Redis) scanRange(keyStart, keyEnd string, keyPrefix string, limit int64) (string, KVs, error) {
	params := []interface{}{
		"pkscanrange", "string_with_value", keyStart, keyEnd,
	}

	if keyPrefix != "" {
		keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"
		params = append(params, "MATCH", keyPrefix)
	}
	if limit > 0 {
		params = append(params, "LIMIT", conv.I64toa(limit))
	}

	_res, err := c.redisClient.Do(c.Ctx, params...).Result()
	if err == redis.Nil {
		return "", nil, nil
	} else if err != nil {
		return "", nil, err
	}

	res, ok := _res.([]interface{})
	if !ok || len(res) <= 1 {
		return "", nil, nil
	}
	nextKey, ok := res[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("scan range returns an invalid next-key")
	}
	_kv, ok := res[1].([]interface{})
	if !ok {
		return "", nil, fmt.Errorf("scan range returns an invalid k/v")
	}
	if len(_kv) <= 0 {
		return nextKey, nil, nil
	}
	// build the kvs
	kvs := KVs{}
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

func (c *Redis) ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, result interface{}) (string, KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis ScanRange: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()
	nextKey, kvs, err := c.scanRange(keyStart, keyEnd, keyPrefix, limit)
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

func (c *Redis) ScanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *KV) error) (string, int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("redis ScanRangeCallback: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()
	nextKey, kvs, err := c.scanRange(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, 0, err
	}

	var read int64 = 0

	for _, kv := range kvs {
		if err != nil {
			return kv.Key, read, err
		}
		read++

		if err = callback(kv); err != nil {
			//遇到错误时, 继续下一个, 根据上面err的判断, 方法会直接返回下一个key, 这样符合nextKey
			continue
		}
	}

	return nextKey, read, err
}
