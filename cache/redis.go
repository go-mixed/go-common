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

func (c *RedisCache) L2() IL2Cache {
	return c.l2Cache
}

func (c *RedisCache) SetNoExpiration(key string, val interface{}) error {
	return c.Set(key, val, 0)
}

func (c *RedisCache) Del(key string) error {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis Del %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.redisClient.Del(c.Ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *RedisCache) Set(key string, val interface{}, expiration time.Duration) error {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.redisClient.Set(c.Ctx, key, text_utils.ToString(val, true), expiration).Result()
	if err != nil {
		return err
	}

	return nil
}

func (c *RedisCache) Get(key string, result interface{}) ([]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis Get %s, %0.6f", key, time.Since(now).Seconds())
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

	if !core_utils.IsInterfaceNil(result) {
		if err := text_utils.JsonUnmarshal(val, result); err != nil {
			c.Logger.Errorf("redis json unmarshal: %s of error: %s", val, err.Error())
			return []byte(val), err
		}
	}
	return []byte(val), nil
}

func (c *RedisCache) MGet(keys []string, result interface{}) (map[string][]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis MGet %v, %0.6f", keys, time.Since(now).Seconds())
	}()
	val, err := c.redisClient.MGet(c.Ctx, keys...).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(val) == 0 { // 一个数据都没找到
		return nil, nil
	}

	kv := map[string][]byte{}
	var vals [][]byte
	for i, v := range val {
		_v, ok := v.(string)
		if !ok {
			kv[keys[i]] = nil
		}
		kv[keys[i]] = []byte(_v)
		vals = append(vals, kv[keys[i]])
	}
	if !core_utils.IsInterfaceNil(result) && len(vals) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(vals, result); err != nil {
			c.Logger.Errorf("redis json unmarshal: %v of error: %s", vals, err.Error())
			return nil, err
		}
	}
	return kv, nil
}

func (c *RedisCache) Keys(keyPattern string) ([]string, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis Keys %s, %0.6f", keyPattern, time.Since(now).Seconds())
	}()
	keyPattern = strings.TrimRight(keyPattern, "*") + "*"
	val, err := c.redisClient.Keys(c.Ctx, keyPattern).Result()
	if err == redis.Nil { // 无此数据
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return val, nil
}

func (c *RedisCache) ScanPrefix(keyPattern string, result interface{}) (map[string][]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis ScanPrefix %s, %0.6f", keyPattern, time.Since(now).Seconds())
	}()
	var cursor uint64
	var err error
	replicateKeys := map[string]bool{}
	var keys []string

	for {
		var _keys []string
		_keys, cursor, err = c.scan(keyPattern, cursor, 10)
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

func (c *RedisCache) ScanPrefixCallback(keyPrefix string, callback func(kv KV) error) (int, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	var cursor uint64
	var err error
	var read int
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
			kv, err := c.MGet(keys, nil)
			if err != nil {
				return read, err
			}
			for k, v := range kv {
				read++

				if err := callback(KV{
					Key:   k,
					Value: v,
				}); err != nil {
					return read, err
				}
			}
		}
	}
}

// Scan 按照的key表达式, 以及游标和页码 返回所有匹配的keys
func (c *RedisCache) scan(keyPattern string, cursor uint64, count int64) ([]string, uint64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis scan %s, cursor %d, count %d, %0.6f", keyPattern, cursor, count, time.Since(now).Seconds())
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

func (c *RedisCache) scanRange(keyStart, keyEnd string, keyPrefix string, limit int) (string, map[string][]byte, error) {
	params := []interface{}{
		"pkscanrange", "string_with_value", keyStart, keyEnd,
	}

	if keyPrefix != "" {
		keyPrefix = strings.TrimRight(keyPrefix, "*") + "*"
		params = append(params, "MATCH", keyPrefix)
	}
	if limit > 0 {
		params = append(params, "LIMIT", conv.Itoa(limit))
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
	// build the map
	kv := map[string][]byte{}
	for i := 0; i < len(_kv); i += 2 {
		k := _kv[i].(string)
		v := _kv[i+1].(string)
		if v == "" {
			kv[k] = nil
		} else {
			kv[k] = []byte(v)
		}
	}
	return nextKey, kv, nil
}

func (c *RedisCache) ScanRange(keyStart, keyEnd string, keyPrefix string, limit int, result interface{}) (string, map[string][]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis ScanRange: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()
	nextKey, kv, err := c.scanRange(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, kv, err
	}

	var vals [][]byte
	for _, v := range kv {
		vals = append(vals, v)
	}

	if !core_utils.IsInterfaceNil(result) && len(vals) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(vals, result); err != nil {
			return "", nil, err
		}
	}

	return nextKey, kv, nil
}

func (c *RedisCache) ScanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int, callback func(kv KV) error) (string, int, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("redis ScanRangeCallback: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()
	nextKey, kv, err := c.scanRange(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, 0, err
	}

	read := 0

	for k, v := range kv {
		if err != nil {
			return k, read, err
		}
		read++

		if err = callback(KV{
			Key:   k,
			Value: v,
		}); err != nil {
			//遇到错误时, 继续下一个, 根据err的判断, 方法会直接返回下一个key, 这样符合nextKey
			continue
		}
	}

	return nextKey, read, err
}
