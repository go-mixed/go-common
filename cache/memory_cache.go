package cache

import (
	ocache "github.com/patrickmn/go-cache"
	"go-common/utils/core"
	"sync"
	"time"
)

var defaultCache *MemoryCache

const (
	// NoExpiration For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// DefaultExpiration For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to NewMemoryCache() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

func init() {
	defaultCache = NewMemoryCache(5*time.Minute, 1*time.Minute)
}

type MemoryCache struct {
	ocache.Cache
	mu sync.Map
}

func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		Cache: *ocache.New(defaultExpiration, cleanupInterval),
		mu:    sync.Map{},
	}
}

func NewFrom(defaultExpiration, cleanupInterval time.Duration, items map[string]ocache.Item) *MemoryCache {
	return &MemoryCache{
		Cache: *ocache.NewFrom(defaultExpiration, cleanupInterval, items),
	}
}

func (c *MemoryCache) SetNoExpiration(key string, value any) {
	c.SetDefault(key, value)
}

func (c *MemoryCache) Remember(k string, expire time.Duration, callback func() (any, error)) (any, error) {
	// 基于Key的锁
	_mu, _ := c.mu.LoadOrStore(k, &sync.Mutex{})
	mu := _mu.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	defer c.mu.Delete(k)

	if v, ok := c.Get(k); ok {
		return v, nil
	} else {
		if v, err := callback(); err != nil {
			return v, err
		} else if v != nil && !core.IsInterfaceNil(v) { // 只有非nil时才能存储
			c.Set(k, v, expire)
			return v, nil
		}
		return nil, nil
	}
}

func SetNoExpiration(k string, v any) {
	defaultCache.Set(k, v, NoExpiration)
}

func Get(k string) (any, bool) {
	return defaultCache.Get(k)
}

func Set(k string, v any, expire time.Duration) {
	defaultCache.Set(k, v, expire)
}

func Remember(k string, expire time.Duration, callback func() (any, error)) (any, error) {
	return defaultCache.Remember(k, expire, callback)
}

func Delete(k string) {
	defaultCache.Delete(k)
}

func DeleteExpired() {
	defaultCache.DeleteExpired()
}

func Flush() {
	defaultCache.Flush()
}
