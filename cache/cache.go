package cache

import (
	ocache "github.com/patrickmn/go-cache"
	"time"
)

var c *ocache.Cache

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

func init() {
	c = ocache.New(5*time.Minute, 1*time.Minute)
}

func Set(k string, v interface{}) {
	c.Set(k, v, NoExpiration)
}

func Get(k string) (interface{}, bool) {
	return c.Get(k)
}

func SetWithExpire(k string, v interface{}, expire time.Duration) {
	c.Set(k, v, expire)
}

func Remember(k string, expire time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	if v, ok := Get(k); ok {
		return v, nil
	} else {
		if v, err := callback(); err != nil {
			return v, err
		} else {
			SetWithExpire(k, v, expire)
			return v, nil
		}
	}
}

func Delete(k string) {
	c.Delete(k)
}

func DeleteExpired() {
	c.DeleteExpired()
}

func Flush() {
	c.Flush()
}
