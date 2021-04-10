package data

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var c *cache.Cache

func init() {
	c = cache.New(5*time.Minute, 1*time.Minute)
}

func Set(k string, v interface{}) {
	c.Set(k, v, cache.NoExpiration)
}

func Get(k string) (interface{}, bool) {
	return c.Get(k)
}

func SetWithExpire(k string, v interface{}, expire time.Duration) {
	c.Set(k, v, expire)
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
