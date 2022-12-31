package cache

import (
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
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
	Get(key string, expire time.Duration, actual any) ([]byte, error)
	MGet(keys []string, expire time.Duration, actual any) (utils.KVs, error)
	Keys(keyPrefix string, expire time.Duration) ([]string, error)
	Delete(keys ...string)

	ScanPrefix(keyPrefix string, expire time.Duration, actual any) (utils.KVs, error)
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

func (l *L2Cache) Get(key string, expire time.Duration, actual any) ([]byte, error) {
	val, err := l.memCache.Remember("get:"+key, expire, func() (any, error) {
		return l.cache.Get(key, nil)
	})

	if err != nil {
		return nil, err
	}

	_val, ok := val.([]byte)
	if ok && _val != nil && !core.IsInterfaceNil(actual) {
		if err := text_utils.JsonUnmarshalFromBytes(_val, actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %s of error: %s", val, err.Error())
			return _val, err
		}
	}

	return _val, nil
}

// MGet 由多个Get构成, 需要维护时, 只需要清理单个Get的缓存即可
// 没有使用 l.Get 是因为避免 IsInterfaceNil 的反射运算浪费时间
func (l *L2Cache) MGet(keys []string, expire time.Duration, actual any) (utils.KVs, error) {
	var _res utils.KVs
	for _, key := range keys {
		if val, err := l.memCache.Remember("get:"+key, expire, func() (any, error) {
			return l.cache.Get(key, nil)
		}); err != nil {
			return nil, err
		} else {
			if _val, ok := val.([]byte); ok {
				_res = _res.Append(key, _val)
			}
		}
	}

	//res, err := l.memCache.Remember("mget:"+text_utils.Md5(strings.Join(keys, "|")), expire, func() (any, error) {
	//	return l.cache.MGet(keys, nil)
	//})
	//if err != nil {
	//	return nil, err
	//}
	//_res, ok := res.(utils.KVs)
	if /*ok &&*/ len(_res) > 0 && !core.IsInterfaceNil(actual) {
		if err := text_utils.JsonListUnmarshalFromBytes(_res.Values(), actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %v of error: %s", _res.Values(), err.Error())
			return nil, err
		}
	}

	return _res, nil
}

func (l *L2Cache) Keys(keyPrefix string, expire time.Duration) ([]string, error) {
	res, err := l.memCache.Remember("keys:"+keyPrefix, expire, func() (any, error) {
		return l.cache.Keys(keyPrefix)
	})
	if err != nil {
		return nil, err
	}

	_res, _ := res.([]string)
	return _res, nil
}

func (l *L2Cache) ScanPrefix(keyPrefix string, expire time.Duration, actual any) (utils.KVs, error) {
	res, err := l.memCache.Remember("scan-prefix:"+keyPrefix, expire, func() (any, error) {
		return l.cache.ScanPrefix(keyPrefix, nil)
	})
	if err != nil {
		return nil, err
	}
	_res, ok := res.(utils.KVs)
	if ok && len(_res) > 0 && !core.IsInterfaceNil(actual) {
		if err := text_utils.JsonListUnmarshalFromBytes(_res.Values(), actual); err != nil {
			l.logger.Errorf("redis json unmarshal: %v of error: %s", _res.Values(), err.Error())
			return nil, err
		}
	}

	return _res, nil
}

func (l *L2Cache) Delete(keys ...string) {
	for _, key := range keys {
		l.memCache.Delete("get:" + key)
		l.memCache.Delete("keys:" + key)
		l.memCache.Delete("scan-prefix:" + key)
	}
}
