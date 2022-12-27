package storage

import (
	"bytes"
	"github.com/pkg/errors"
	"go-common/utils"
	"go-common/utils/core"
	"go-common/utils/text"
	bolt "go.etcd.io/bbolt"
	"time"
)

type Bolt struct {
	DB *bolt.DB

	logger     utils.ILogger
	decodeFunc func([]byte, any) error
	encodeFunc func(any) ([]byte, error)
}

type BoltBucket struct {
	*Bolt
	bucket []byte
}

func NewBolt(path string, logger utils.ILogger) (*Bolt, error) {
	db, err := bolt.Open(path, 0o664, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, errors.Errorf("open bolt file \"%s\" error: %w", path, err)
	}

	return &Bolt{
		DB:     db,
		logger: logger,

		encodeFunc: text_utils.JsonMarshalToBytes,
		decodeFunc: text_utils.JsonUnmarshalFromBytes,
	}, nil
}

func (b *Bolt) SetEncodeFunc(encodeFunc func(any) ([]byte, error)) *Bolt {
	b.encodeFunc = encodeFunc
	return b
}

func (b *Bolt) SetDecodeFunc(decodeFunc func([]byte, any) error) *Bolt {
	b.decodeFunc = decodeFunc
	return b
}

func (b *Bolt) Bucket(bucket string) *BoltBucket {
	return &BoltBucket{
		Bolt:   b,
		bucket: []byte(bucket),
	}
}

func (b *Bolt) Close() error {
	return b.DB.Close()
}

func (b *BoltBucket) Batch(callback func(*bolt.Bucket) error) error {
	return b.DB.Batch(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(b.bucket)
		if err != nil {
			b.logger.Errorf("bolt bucket %s error: %s", b.bucket, err.Error())
			return errors.WithStack(err)
		}

		return errors.WithStack(callback(bucket))
	})
}

func (b *BoltBucket) View(callback func(*bolt.Bucket) error) error {
	return b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			return nil
		}
		return errors.WithStack(callback(bucket))
	})
}

func (b *BoltBucket) Update(callback func(*bolt.Bucket) error) error {
	return b.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(b.bucket)
		if err != nil {
			b.logger.Errorf("[Bolt]bucket %s error: %s", b.bucket, err.Error())
			return err
		}
		return errors.WithStack(callback(bucket))
	})
}

func (b *BoltBucket) Get(key string, actual any) ([]byte, error) {
	var buf []byte
	err := b.View(func(bucket *bolt.Bucket) error {
		if buf = bucket.Get([]byte(key)); buf != nil && !core.IsInterfaceNil(actual) {
			if err := b.decodeFunc(buf, actual); err != nil {
				b.logger.Errorf("[Bolt]Get data and decode error: %s", err.Error())
				return errors.WithStack(err)
			}
		}
		return nil
	})

	return buf, err
}

func (b *BoltBucket) Keys() ([]string, error) {
	var res []string
	err := b.View(func(bucket *bolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			res = append(res, string(k))
			return nil
		})
	})
	return res, err
}

func (b *BoltBucket) Values() ([]string, error) {
	var res []string
	err := b.View(func(bucket *bolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			res = append(res, string(v))
			return nil
		})
	})
	return res, err
}

func (b *BoltBucket) GetAll(actual any) (utils.KVs, error) {
	kvs := utils.KVs{}
	err := b.View(func(bucket *bolt.Bucket) error {
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			kvs = kvs.Append(string(k), v)
		}
		if actual != nil && !core.IsInterfaceNil(actual) {
			if err := text_utils.ListDecode(b.decodeFunc, kvs.Values(), actual); err != nil {
				b.logger.Errorf("[Bolt]Get data and decode error: %s", err.Error())
				return errors.WithStack(err)
			}
		}
		return nil
	})
	return kvs, err
}

func (b *BoltBucket) Set(key string, value any) error {
	return b.Update(func(bucket *bolt.Bucket) error {
		buf, err := b.encodeFunc(value)
		if err != nil {
			return errors.WithMessagef(err, "[Bolt]Set data and encode error")
		}
		if err = bucket.Put([]byte(key), buf); err != nil {
			b.logger.Errorf("[Bolt]Set error: %s", err.Error())
		}
		return errors.WithStack(err)
	})
}

func (b *BoltBucket) Delete(key string) error {
	return b.Update(func(bucket *bolt.Bucket) error {
		err := bucket.Delete([]byte(key))
		if err != nil {
			b.logger.Errorf("[Bolt]Set error: %s", err.Error())
		}
		return errors.WithStack(err)
	})
}

func (b *BoltBucket) Count() int {
	var count int
	if b.bucket != nil {
		if err := b.DB.View(func(tx *bolt.Tx) error {
			if bucket := tx.Bucket(b.bucket); bucket != nil {
				stats := bucket.Stats()
				count = stats.KeyN
			}
			return nil
		}); err != nil {
			b.logger.Errorf("[Bolt]read bucket \"%s\" count error: %s", b.bucket, err.Error())
		}
	}
	return count
}

func (b *BoltBucket) Clear() error {
	if b.bucket != nil {
		return b.DB.Update(func(tx *bolt.Tx) error {
			if bucket := tx.Bucket(b.bucket); bucket != nil {
				return tx.DeleteBucket(b.bucket)
			}
			return nil
		})
	}
	return nil
}

// FindLte 查找 等于key 或 小于key的上一项 Less than and equal
// 注意: 返回的key可能和需要查找key并不相似
// 返回 key value 错误
func (b *BoltBucket) FindLte(key string, actual any) (utils.KV, error) {
	var buf []byte
	err := b.View(func(bucket *bolt.Bucket) error {
		_key := []byte(key)
		cursor := bucket.Cursor()
		_key, buf = cursor.Seek(_key)
		if !bytes.Equal(_key, []byte(key)) { // 没有找到等于的key, 会返回下一个符合要求的项
			_key, buf = cursor.Prev()
			if _key != nil {
				key = string(_key)
			} else {
				key = ""
			}
		}

		if buf != nil && !core.IsInterfaceNil(actual) {
			if err := b.decodeFunc(buf, actual); err != nil {
				b.logger.Errorf("[Bolt]FindLte data and decode error: %s", err.Error())
				return errors.WithStack(err)
			}
		}

		return nil
	})

	return utils.KV{Key: key, Value: buf}, err
}

func (b *BoltBucket) FindLt(key string, actual any) (utils.KV, error) {
	var buf []byte
	err := b.View(func(bucket *bolt.Bucket) error {
		_key := []byte(key)
		cursor := bucket.Cursor()
		_key, buf = cursor.Seek(_key)
		if bytes.Compare(_key, []byte(key)) >= 0 { // 没有找到等于的key, 会返回下一个符合要求的项
			_key, buf = cursor.Prev()
			if _key != nil {
				key = string(_key)
			} else {
				key = ""
			}
		}

		if buf != nil && !core.IsInterfaceNil(actual) {
			err := b.decodeFunc(buf, actual)
			if err != nil {
				b.logger.Errorf("[Bolt]FindLte data and decode error: %s", err.Error())
				return errors.WithStack(err)
			}
		}

		return nil
	})

	return utils.KV{Key: key, Value: buf}, err
}

// FindGte 查找 等于key的 或 大于key的下一项 Greater than and equal
// 注意: 返回的key可能和需要查找key并不相似
// 返回 key value 错误
func (b *BoltBucket) FindGte(key string, actual any) (utils.KV, error) {
	var buf []byte
	err := b.View(func(bucket *bolt.Bucket) error {
		_key := []byte(key)
		cursor := bucket.Cursor()
		_key, buf = cursor.Seek(_key) // 如果没有找到key, 会返回下一项, 如果到了结尾 _key/buf为nil
		key = string(_key)

		if buf != nil && !core.IsInterfaceNil(actual) {
			if err := b.decodeFunc(buf, actual); err != nil {
				b.logger.Errorf("[Bolt]FindGte data and decode error: %s", err.Error())
				return errors.WithStack(err)
			}
		}

		return nil
	})

	return utils.KV{Key: key, Value: buf}, err
}

func (b *BoltBucket) Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	_keyPrefix := []byte(keyPrefix)
	_keyStart := []byte(keyStart)
	_keyEnd := []byte(keyEnd)
	if bytes.Compare(_keyStart, _keyEnd) > 0 {
		return "", nil, errors.Errorf("[Bolt]error key range, \"keyStart\" must less than \"keyEnd\"")
	}

	var _nextKey []byte
	kvs := utils.KVs{}
	if err := b.View(func(bucket *bolt.Bucket) error {
		cursor := bucket.Cursor()
		var i int64 = 0
		var k []byte
		var v []byte
		for k, v = cursor.Seek(_keyStart); i < limit && k != nil && bytes.HasPrefix(k, _keyPrefix) && bytes.Compare(k, _keyEnd) < 0; k, v = cursor.Next() {
			kvs = kvs.Append(string(k), v)
			i++
		}
		if i > 0 && i == limit {
			_nextKey = k
		}
		return nil
	}); err != nil {
		return "", nil, err
	}

	return string(_nextKey), kvs, nil
}

func (b *BoltBucket) RevRange(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	_keyPrefix := []byte(keyPrefix)
	_keyStart := []byte(keyStart)
	_keyEnd := []byte(keyEnd)
	if bytes.Compare(_keyStart, _keyEnd) < 0 {
		return "", nil, errors.Errorf("error key range, \"keyStart\" must greater than \"keyEnd\"")
	}

	var _prevKey []byte
	kvs := utils.KVs{}
	if err := b.View(func(bucket *bolt.Bucket) error {
		cursor := bucket.Cursor()
		var i int64 = 0
		var k []byte
		var v []byte

		k, v = cursor.Seek(_keyStart)
		// seek是找到匹配项, 或相近项的下一个, 如果第一次找不到匹配项 需要尝试prev
		if k == nil { // 找不到先返回last
			k, v = cursor.Last()
		} else if bytes.Compare(k, _keyStart) > 0 {
			k, v = cursor.Prev()
		}
		if k == nil || bytes.Compare(k, _keyStart) > 0 {
			return nil
		}

		for ; i < limit && k != nil && bytes.HasPrefix(k, _keyPrefix) && bytes.Compare(k, _keyEnd) > 0; k, v = cursor.Prev() {
			kvs = kvs.Append(string(k), v)
			i++
		}
		if i > 0 && i == limit {
			_prevKey = k
		}
		return nil
	}); err != nil {
		return "", nil, err
	}

	return string(_prevKey), kvs, nil
}
