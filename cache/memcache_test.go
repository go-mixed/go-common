package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := NewMemoryCache(DefaultExpiration, 1*time.Minute)
	c.SetNoExpiration("123", "456")
	res, _ := c.Get("123")
	t.Logf("cache 123: %s", res)
}
