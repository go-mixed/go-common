package cache

import (
	"go-common/utils"
	"strings"
	"time"
)

type L2Cache struct {
	memCache *MemoryCache
	cache    ICache
	logger   utils.ILogger
}

type L2Result struct {
	ok   bool
	json []byte
}

type IL2Cache interface {
	Get(key string, expire time.Duration, actual interface{}) ([]byte, error)
	MGet(keys []string, expire time.Duration, actual interface{}) (map[string][]byte, error)
	Keys(keyPrefix string, expire time.Duration) ([]string, error)
	Delete(keys ...string)

	ScanPrefix(keyPrefix string, expire time.Duration, actual interface{}) (map[string][]byte, error)
}

func NewL2Cache(
	cache ICache,
	logger utils.ILogger,
) *L2Cache {
	return &L2Cache{
		cache:    cache,
		memCache: NewMemoryCache(5*time.Minute, 1*time.Minute),
		logger:   logger,
	}
}

func (l *L2Cache) Get(key string, expire time.Duration, actual interface{}) ([]byte, error) {
	val, err := l.memCache.Remember("get:"+key, expire, func() (interface{}, error) {
		return l.cache.Get(key, nil)
	})

	if err != nil {
		return nil, err
	}

	_val, ok := val.([]byte)
	if ok && _val != nil && !utils.IsInterfaceNil(actual) {
		if err := utils.JsonUnmarshalFromBytes(_val, actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %s of error: %s", val, err.Error())
			return _val, err
		}
	}

	return _val, nil
}

func (l *L2Cache) MGet(keys []string, expire time.Duration, actual interface{}) (map[string][]byte, error) {
	res, err := l.memCache.Remember("mget:"+utils.Md5(strings.Join(keys, "|")), expire, func() (interface{}, error) {
		return l.cache.MGet(keys, nil)
	})
	if err != nil {
		return nil, err
	}
	_res, ok := res.(map[string][]byte)
	if ok && len(_res) > 0 && !utils.IsInterfaceNil(actual) {
		var _vals [][]byte
		for _, v := range _res {
			_vals = append(_vals, v)
		}
		if err := utils.JsonListUnmarshalFromBytes(_vals, actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %v of error: %s", _vals, err.Error())
			return nil, err
		}
	}

	return _res, nil
}

func (l *L2Cache) Keys(keyPrefix string, expire time.Duration) ([]string, error) {
	res, err := l.memCache.Remember("keys:"+keyPrefix, expire, func() (interface{}, error) {
		return l.cache.Keys(keyPrefix)
	})
	if err != nil {
		return nil, err
	}

	_res, _ := res.([]string)
	return _res, nil
}

func (l *L2Cache) Delete(keys ...string) {
	for _, key := range keys {
		l.memCache.Delete("key:" + key)
	}
}

func (l *L2Cache) ScanPrefix(keyPrefix string, expire time.Duration, actual interface{}) (map[string][]byte, error) {
	res, err := l.memCache.Remember("scan-prefix:"+keyPrefix, expire, func() (interface{}, error) {
		return l.cache.ScanPrefix(keyPrefix, nil)
	})
	if err != nil {
		return nil, err
	}
	_res, ok := res.(map[string][]byte)
	if ok && !utils.IsInterfaceNil(actual) {
		var _vals [][]byte
		for _, v := range _res {
			_vals = append(_vals, v)
		}
		if err := utils.JsonListUnmarshalFromBytes(_vals, actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %v of error: %s", _vals, err.Error())
			return nil, err
		}
	}

	return _res, nil
}