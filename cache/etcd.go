package cache

import (
	"bytes"
	"go-common/utils/core"
	"go-common/utils/text"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"strings"
	"time"
)

type Etcd struct {
	Cache
	client  *clientv3.Client
	l2Cache *L2Cache
}

func (c *Etcd) SetNoExpiration(key string, val interface{}) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.client.Put(c.Ctx, key, text_utils.ToString(val, true))
	if err != nil {
		return err
	}

	return nil
}

func (c *Etcd) Del(key string) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd Del %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.client.Delete(c.Ctx, key)
	if err != nil {
		return err
	}

	return nil
}

func (c *Etcd) Set(key string, val interface{}, expiration time.Duration) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	ttl := int64(expiration.Seconds())
	if ttl <= 0 {
		return c.SetNoExpiration(key, val)
	}

	lease := clientv3.NewLease(c.client)
	response, err := lease.Grant(c.Ctx, ttl)
	if err != nil {
		return err
	}

	_, err = c.client.Put(c.Ctx, key, text_utils.ToString(val, true), clientv3.WithLease(response.ID))
	if err != nil {
		return err
	}

	return nil
}

// LastRevision etcd的最后版本号
func (c *Etcd) LastRevision() int64 {
	response, err := c.client.Get(c.Ctx, "\x00", clientv3.WithLastRev()...)
	if err != nil {
		return -1
	}
	return response.Header.GetRevision()
}

func (c *Etcd) Get(key string, actual interface{}) ([]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("etcd Get %s, %0.6f", key, time.Since(now).Seconds())
	}()
	kv := clientv3.NewKV(c.client)
	response, err := kv.Get(c.Ctx, key, clientv3.WithLimit(1))
	if err != nil {
		c.Logger.Debugf("etcd error of key %s", key, err.Error())
		return nil, err
	} else if response.Count == 0 {
		c.Logger.Debugf("etcd key not exists: %s", key)
		return nil, nil
	} else if len(response.Kvs[0].Value) == 0 {
		c.Logger.Debugf("etcd empty value of key %s", key)
		return nil, nil
	}

	var val = response.Kvs[0].Value
	if !core.IsInterfaceNil(actual) {
		if err := text_utils.JsonUnmarshalFromBytes(val, actual); err != nil {
			c.Logger.Errorf("etcd json unmarshal: %s of error: %s", val, err.Error())
			return []byte(val), err
		}
	}
	return val, nil
}

func (c *Etcd) MGet(keys []string, actual interface{}) (KVs, error) {
	kv := clientv3.NewKV(c.client)

	kvs := KVs{}
	for _, key := range keys {
		response, err := kv.Get(c.Ctx, key, clientv3.WithLimit(1))
		if err != nil {
			return nil, err
		} else if response.Count == 0 {
			kvs = kvs.Append(key, nil)
		} else {
			kvs = kvs.Append(key, response.Kvs[0].Value)
		}
	}

	if !core.IsInterfaceNil(actual) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), actual); err != nil {
			c.Logger.Errorf("redis json unmarshal: %v of error: %s", kvs.Values(), err.Error())
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Etcd) Keys(keyPrefix string) ([]string, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd Keys %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	kv := clientv3.NewKV(c.client)
	response, err := kv.Get(c.Ctx, keyPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}
	var keys []string
	for i := range response.Kvs {
		keys = append(keys, string(response.Kvs[i].Key))
	}

	return keys, nil
}

// ScanWithRev 得到指定版本范围内的 keyPrefix 列表
func (c *Etcd) ScanWithRev(keyPrefix string, minRev, maxRev int64) (*clientv3.GetResponse, error) {
	kv := clientv3.NewKV(c.client)
	return kv.Get(c.Ctx, keyPrefix, clientv3.WithPrefix(), clientv3.WithMinModRev(minRev), clientv3.WithMaxModRev(maxRev))
}

func (c *Etcd) scanRange(keyStart, keyEnd string, keyPrefix string, limit int64) (string, KVs, error) {
	kv := clientv3.NewKV(c.client)

	kvs := KVs{}

	response, err := kv.Get(c.Ctx, keyStart, clientv3.WithFromKey(), clientv3.WithRange(keyEnd), clientv3.WithLimit(limit+1)) // 多取1个是为了返回最后一个为nextKey
	if err != nil {
		return "", nil, err
	} else if response.Count == 0 {
		return "", nil, nil
	}
	var i int64
	count := core.If(limit-1 < response.Count, limit-1, response.Count).(int64)
	for i = 0; i < count; i++ {
		key := string(response.Kvs[i].Key)
		if strings.HasPrefix(key, keyPrefix) {
			kvs = kvs.Append(key, response.Kvs[i].Value)
		}
	}

	// 总数不足limit个 则说明已经没nextKey
	if int64(len(kvs)) < limit {
		return "", kvs, nil
	} else {
		return string(response.Kvs[limit].Key), kvs, nil // 取第limit+1个作为nextKey
	}

}

func (c *Etcd) ScanPrefix(keyPrefix string, result interface{}) (KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd ScanPrefix %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	kvs := KVs{}

	var keyStart = keyPrefix
	var keyEnd = clientv3.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs KVs
	for {
		nextKey, _kvs, err = c.scanRange(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return nil, err
		}
		kvs = kvs.Add(_kvs)
		if nextKey == "" {
			break
		}
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.JsonListUnmarshalFromBytes(kvs.Values(), result); err != nil {
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Etcd) ScanPrefixCallback(keyPrefix string, callback func(kv *KV) error) (int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	var keyStart = keyPrefix
	var keyEnd = clientv3.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs KVs
	var read int64
	for {
		nextKey, _kvs, err = c.scanRange(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return read, err
		}

		for _, kv := range _kvs {
			read++
			if err := callback(kv); err != nil {
				return read, err
			}
		}

		if nextKey == "" {
			return read, nil
		}
	}
}

// ScanRangeRaw 得到keyStart~keyEnd范围内，并且符合keyPrefix的数据, limit <= 0则表示取无限个
func (c *Etcd) ScanRangeRaw(keyStart string, keyEnd string, keyPrefix string, limit int64) (*clientv3.GetResponse, error) {
	kv := clientv3.NewKV(c.client)

	if limit < 0 {
		limit = 0
	}

	response, err := kv.Get(c.Ctx, keyStart, clientv3.WithFromKey(), clientv3.WithRange(keyEnd), clientv3.WithLimit(limit))
	if err != nil {
		return nil, err
	}

	if keyPrefix == "" {
		return response, nil
	}

	var kvs []*mvccpb.KeyValue

	var _keyPrefix = []byte(keyPrefix)
	for i := range response.Kvs {
		if bytes.HasPrefix(response.Kvs[i].Key, _keyPrefix) {
			kvs = append(kvs, response.Kvs[i])
		}
	}

	response.Count = int64(len(kvs))

	return response, nil
}

func (c *Etcd) ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, result interface{}) (string, KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd ScanRange: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
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

func (c *Etcd) ScanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *KV) error) (string, int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("etcd ScanRangeCallback: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
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
