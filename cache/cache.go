package cache

import (
	ocache "github.com/patrickmn/go-cache"
	"time"
	"sync"
)

var defaultCache *Cache

const (
	// NoExpiration For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// DefaultExpiration For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

func init() {
	defaultCache = New(5*time.Minute, 1*time.Minute)
}

type Cache struct {
	ocache.Cache
	mu sync.Map
}

func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		Cache: *ocache.New(defaultExpiration, cleanupInterval),
		mu: sync.Map{},
	}
}

func NewFrom(defaultExpiration, cleanupInterval time.Duration, items map[string]ocache.Item) *Cache {
	return &Cache{
		Cache: *ocache.NewFrom(defaultExpiration, cleanupInterval, items),
	}
}

func (c *Cache) Remember(k string, expire time.Duration, callback func() (interface{}, error)) (interface{}, error) {
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
		} else {
			c.Set(k, v, expire)
			return v, nil
		}
	}
}


func SetNoExpiration(k string, v interface{}) {
	defaultCache.Set(k, v, NoExpiration)
}

func Get(k string) (interface{}, bool) {
	return defaultCache.Get(k)
}

func Set(k string, v interface{}, expire time.Duration) {
	defaultCache.Set(k, v, expire)
}

func Remember(k string, expire time.Duration, callback func() (interface{}, error)) (interface{}, error) {
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
