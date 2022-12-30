package storage

import (
	"bytes"
	"github.com/pkg/errors"
	"go-common/utils"
	"go-common/utils/core"
	"go-common/utils/text"
	bolt "go.etcd.io/bbolt"
	"strings"
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

var ErrForEachBreak = errors.New("for each break")

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

// Batch 批量操作（事务），注意：使用Batch的写入操作会有延迟（MaxBatchDelay），其它事务会出现幻读
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

// View 只读操作
func (b *BoltBucket) View(callback func(*bolt.Bucket) error) error {
	return b.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.bucket)
		if bucket == nil {
			return nil
		}
		return errors.WithStack(callback(bucket))
	})
}

// Update 修改操作，注意：和 Batch 不同的是，Update中写入操作是实时的
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

// ForEach 遍历所有kv，可以对bolt进行修改
func (b *BoltBucket) ForEach(callback func(bucket *bolt.Bucket, kv *utils.KV) error) (int64, error) {
	var i int64
	err := b.Update(func(bucket *bolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			i++
			err := callback(bucket, utils.NewKV(string(k), v))
			return errors.WithStack(err)
		})
	})
	return i, err
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

// BatchDeleteRange 延迟批量删除：从keyStart（含）删除到keyEnd（含），并且需要匹配前缀keyPrefix
//
//	keyStart为空表示第一个key，keyEnd为空表示最后一个key keyPrefix为空表示不筛选前缀
//	注意：由于使用的是Batch，所以删除有延时，其它事务会出现幻读
func (b *BoltBucket) BatchDeleteRange(keyStart string, keyEnd string, keyPrefix string) (int64, error) {
	_, n, err := b.rangeCallback(b.Batch, keyStart, keyEnd, keyPrefix, -1, func(bucket *bolt.Bucket, kv *utils.KV) error {
		err := bucket.Delete([]byte(kv.Key))
		if err != nil {
			b.logger.Errorf("[Bolt]deleting \"%s\" of bucket: \"%s\" error: %s", kv.Key, b.bucket, err.Error())
		}
		return err
	})
	return n, err
}

// DeleteRange 实时批量删除（性能会降低）: 从keyStart（含）删除到keyEnd（含），并且需要匹配前缀keyPrefix
//
//	keyStart为空表示bucket第一个，keyEnd为空表示直到bucket最后一个，keyPrefix为空表示不筛选前缀
func (b *BoltBucket) DeleteRange(keyStart string, keyEnd string, keyPrefix string) (int64, error) {
	_, n, err := b.rangeCallback(b.Update, keyStart, keyEnd, keyPrefix, -1, func(bucket *bolt.Bucket, kv *utils.KV) error {
		err := bucket.Delete([]byte(kv.Key))
		if err != nil {
			b.logger.Errorf("[Bolt]deleting \"%s\" of bucket: \"%s\" error: %s", kv.Key, b.bucket, err.Error())
		}
		return err
	})
	return n, err
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

// Range 返回指定范围内的所有kv，从keyStart（含）到keyEnd（含），并符合前缀keyPrefix，以及数量在小于等于limit，limit为-1表示不限
//
//	返回：下一个key，符合要求的kvs，错误
func (b *BoltBucket) Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	kvs := utils.KVs{}

	nextKey, _, err := b.rangeCallback(b.View, keyStart, keyEnd, keyPrefix, limit, func(bucket *bolt.Bucket, kv *utils.KV) error {
		kvs = append(kvs, kv)
		return nil
	})

	return nextKey, kvs, err
}

// RevRange 【反转】返回指定范围内的所有kv，从keyStart（含）到keyEnd（含），并符合前缀keyPrefix，以及数量在小于等于limit，limit为-1表示不限
//
//	返回：上一个key，符合要求的kvs，错误
func (b *BoltBucket) RevRange(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	if limit == 0 {
		return "", nil, nil
	}
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

// RangeCallback 按范围执行回调：从keyStart（含）循环到keyEnd（含），并且匹配前缀keyPrefix，以及数量小于等于limit
// keyStart为空表示第一个key，keyEnd为空表示最后一个key，keyPrefix为空表示不筛选前缀，limit为-1表示数量不限
func (b *BoltBucket) RangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(bucket *bolt.Bucket, kv *utils.KV) error) (string, int64, error) {
	return b.rangeCallback(b.Update, keyStart, keyEnd, keyPrefix, limit, callback)
}

// keyStart为空表示第一个key，keyEnd为空表示最后一个key，keyPrefix为空表示不筛选前缀，limit为-1表示数量不限
// fn为b.View、b.Update、b.Batch，callback为每一次循环的回调
func (b *BoltBucket) rangeCallback(fn func(callback func(*bolt.Bucket) error) error, keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(bucket *bolt.Bucket, kv *utils.KV) error) (string, int64, error) {
	if limit == 0 {
		return "", 0, nil
	}
	_keyPrefix := []byte(keyPrefix)
	_keyStart := []byte(keyStart)
	_keyEnd := []byte(keyEnd)
	var i int64 = 0

	var realKeyStart []byte
	var realKeyEnd []byte
	var _nextKey string

	if keyStart != "" && keyEnd != "" && strings.Compare(keyStart, keyEnd) > 0 {
		return _nextKey, i, errors.Errorf("[Bolt]range error, \"keyStart\" must less than \"keyEnd\" if they both defined")
	}

	err := fn(func(bucket *bolt.Bucket) error {
		cursor := bucket.Cursor()
		var k []byte
		var v []byte
		if keyStart != "" { // 搜寻起始值
			k, v = cursor.Seek(_keyStart)
		} else { // 否则从开头开始
			k, v = cursor.First()
		}
		realKeyStart = k
		for ; k != nil; k, v = cursor.Next() {
			if keyPrefix != "" && !bytes.HasPrefix(k, _keyPrefix) { // 前缀不符
				continue
			} else if keyEnd != "" && bytes.Compare(k, _keyEnd) > 0 { // 超过keyEnd
				break
			}

			i++
			realKeyEnd = k

			if err := callback(bucket, utils.NewKV(string(k), v)); err != nil {
				//b.logger.Errorf("[Bolt]foreach \"%s\" of bucket: \"%s\" error: %s", k, b.bucket, err.Error())
				return err
			}
			//b.logger.Debugf("[Bolt]foreach \"%s\" of bucket: \"%s\"", k, b.bucket)

			// 超过limit
			if limit > 0 && i >= limit {
				break
			}
		}

		_nextKey = string(k)
		return nil
	})

	b.logger.Debugf("[Bolt]foreach %d items of bucket: \"%s\", from \"%s\" to \"%s\"", i, b.bucket, realKeyStart, realKeyEnd)
	return _nextKey, i, err
}
