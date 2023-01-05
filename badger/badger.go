package badger

import (
	"bytes"
	"github.com/dgraph-io/badger/v3"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"path/filepath"
	"strings"
	"sync"
)

type Badger struct {
	baseDir    string
	logger     utils.ILogger
	decodeFunc text_utils.DecoderFunc
	encodeFunc text_utils.EncoderFunc

	buckets sync.Map
	options badger.Options
}

type BadgerBucket struct {
	bucket string
	b      *Badger
	db     *badger.DB
}

func NewBadger(path string, logger utils.ILogger, writeSync bool) *Badger {
	return &Badger{
		baseDir:    path,
		logger:     logger,
		encodeFunc: text_utils.JsonMarshalToBytes,
		decodeFunc: text_utils.JsonUnmarshalFromBytes,

		buckets: sync.Map{},
		options: badger.DefaultOptions("").WithLogger(iLogger{logger}).WithLoggingLevel(badger.DEBUG).WithSyncWrites(writeSync),
	}
}

func (p *Badger) SetEncodeFunc(encodeFunc text_utils.EncoderFunc) *Badger {
	p.encodeFunc = encodeFunc
	return p
}

func (p *Badger) SetDecodeFunc(decodeFunc text_utils.DecoderFunc) *Badger {
	p.decodeFunc = decodeFunc
	return p
}

func (p *Badger) Bucket(name string) *BadgerBucket {
	bucket, ok := p.buckets.Load(name)
	if !ok {
		dir := filepath.Join(p.baseDir, name)
		db, err := badger.Open(p.options.WithDir(dir).WithValueDir(dir))
		if err != nil {
			panic(err)
		}
		bucket = &BadgerBucket{bucket: name, b: p, db: db}
		p.buckets.Store(name, bucket)
	}

	return bucket.(*BadgerBucket)
}

func (p *Badger) BucketNotCreate(name string) *BadgerBucket {
	bucket, ok := p.buckets.Load(name)
	if ok {
		return bucket.(*BadgerBucket)
	}

	return nil
}

func (p *Badger) DeleteBucket(name string) error {
	var err error
	if bucket := p.BucketNotCreate(name); bucket != nil {
		err = bucket.db.Close()
		p.buckets.Delete(name)
	}
	return err
}

func (p *Badger) Close() error {
	var err error
	p.buckets.Range(func(key, value any) bool {
		err = multierr.Append(err, value.(*BadgerBucket).Close())
		return true
	})

	return err
}

func (b *BadgerBucket) Close() error {
	return b.db.Close()
}

func (b *BadgerBucket) Set(key string, val any) error {
	buf, err := b.b.encodeFunc(val)
	if err != nil {
		return errors.WithStack(err)
	}

	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), buf)
	})
}

func (b *BadgerBucket) Get(key string, actual any) ([]byte, error) {
	var buf []byte
	if err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return errors.WithStack(err)
		}
		_, buf, err = b.getKV(item)
		return errors.WithStack(err)
	}); err != nil {
		return nil, err
	}

	if err := b.b.decodeFunc(buf, actual); err != nil {
		b.b.logger.Errorf("[Badger]Get data and decode error: %s", err.Error())
		return nil, errors.WithStack(err)
	}

	return buf, nil
}

func (b *BadgerBucket) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b *BadgerBucket) getKV(item *badger.Item) ([]byte, []byte, error) {
	key := core.CopyFrom(item.Key())
	var val []byte
	err := item.Value(func(v []byte) error {
		val = core.CopyFrom(v)
		return nil
	})
	return key, val, err
}

func (b *BadgerBucket) ForEach(callback func(txn *badger.Txn, kv *utils.KV) error) (int64, error) {
	_, n, err := b.rangeCallback(b.db.View, "", "", "", -1, func(txn *badger.Txn, kv *utils.KV) error {
		return callback(txn, kv)
	})
	return n, err
}

// DeleteRange 删除范围keyStart（含）~keyEnd（不含）
func (b *BadgerBucket) DeleteRange(keyStart string, keyEnd string, keyPrefix string) (int64, error) {
	_, n, err := b.rangeCallback(b.db.Update, keyStart, keyEnd, keyPrefix, -1, func(txn *badger.Txn, kv *utils.KV) error {
		return txn.Delete([]byte(kv.Key))
	})
	return n, err
}

// Range 返回指定范围内的所有kv，从keyStart（含）到keyEnd（含），并符合前缀keyPrefix，以及数量在小于等于limit，limit为-1表示不限
//
//	返回：下一个key，符合要求的kvs，错误
func (b *BadgerBucket) Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	kvs := utils.KVs{}

	nextKey, _, err := b.rangeCallback(b.db.View, keyStart, keyEnd, keyPrefix, limit, func(txn *badger.Txn, kv *utils.KV) error {
		kvs = append(kvs, kv)
		return nil
	})

	return nextKey, kvs, err
}

// RangeCallback 按范围执行回调：从keyStart（含）循环到keyEnd（含），并且匹配前缀keyPrefix，以及数量小于等于limit
// keyStart为空表示第一个key，keyEnd为空表示最后一个key，keyPrefix为空表示不筛选前缀，limit为-1表示数量不限
func (b *BadgerBucket) RangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(txn *badger.Txn, kv *utils.KV) error) (string, int64, error) {
	return b.rangeCallback(b.db.Update, keyStart, keyEnd, keyPrefix, limit, callback)
}

// keyStart为空表示第一个key，keyEnd为空表示最后一个key，keyPrefix为空表示不筛选前缀，limit为-1表示数量不限
// fn为b.View、b.Update、b.Batch，callback为每一次循环的回调
func (b *BadgerBucket) rangeCallback(fn func(func(txn *badger.Txn) error) error, keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(txn *badger.Txn, kv *utils.KV) error) (string, int64, error) {
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
		return _nextKey, i, errors.Errorf("[Badger]range error, \"keyStart\" must less than \"keyEnd\" if they both defined")
	}

	err := fn(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		if keyStart != "" {
			it.Seek(_keyStart)
		}
		// 获取真实开始的key
		if it.Valid() {
			realKeyStart, _, _ = b.getKV(it.Item())
		}

		var key []byte
		var val []byte
		var err error

		for ; ; it.Next() {
			if it.Valid() {
				key, val, err = b.getKV(it.Item())
			} else {
				key = nil // 下面的_nextKey取到正确的值
				val = nil
				break
			}
			if err != nil {
				return errors.WithStack(err)
			} else if keyPrefix != "" && !bytes.HasPrefix(key, _keyPrefix) { // 前缀不符
				continue
			} else if keyEnd != "" && bytes.Compare(key, _keyEnd) > 0 { // 超过keyEnd
				break
			}

			i++
			realKeyEnd = core.CopyFrom(key) // GC 后key会被清空，必须Copy

			if err = callback(txn, utils.NewKV(string(key), core.CopyFrom(val))); err != nil {
				//b.logger.Errorf("[Bolt]foreach \"%s\" of bucket: \"%s\" error: %s", k, b.bucket, err.Error())
				return err
			}
			//b.logger.Debugf("[Badger]foreach \"%s\" of bucket: \"%s\"", k, b.bucket)

			// 超过limit
			if limit > 0 && i >= limit {
				break
			}
		}
		_nextKey = string(key)

		return nil
	})

	if i > 0 { // realKeyStart和realKeyEnd为空时 会panic
		b.b.logger.Debugf("[Badger]foreach %d items of bucket: \"%s\", from \"%s\" to \"%s\"", i, b.bucket, realKeyStart, realKeyEnd)
	}

	return _nextKey, i, err
}

func (b *BadgerBucket) Count() int64 {
	var i int64
	_ = b.db.View(func(txn *badger.Txn) error {
		options := badger.DefaultIteratorOptions
		options.PrefetchValues = false

		it := txn.NewIterator(options)
		defer it.Close()
		for ; it.Valid(); it.Next() {

			i++
		}
		return nil
	})

	return i
}

func (b *BadgerBucket) Clear() error {
	return b.b.DeleteBucket(b.bucket)
}
