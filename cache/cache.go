package cache

import (
	"context"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
)

type Cache struct {
	Ctx     context.Context
	Logger  utils.ILogger
	L2Cache *L2Cache

	DecodeFunc text_utils.DecoderFunc
	EncodeFunc text_utils.EncoderFunc
}

func (c *Cache) L2() utils.IMemKV {
	return c.L2Cache
}

type RangeFunc func(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error)

func (c *Cache) SetEncodeFunc(encodeFunc text_utils.EncoderFunc) *Cache {
	c.EncodeFunc = encodeFunc
	return c
}

func (c *Cache) SetDecodeFunc(decodeFunc text_utils.DecoderFunc) *Cache {
	c.DecodeFunc = decodeFunc
	return c
}

func (c *Cache) ScanRangeFn(keyStart, keyEnd string, keyPrefix string, limit int64, result any, rangeFunc RangeFunc) (string, utils.KVs, error) {
	nextKey, kvs, err := rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, kvs, err
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.ListDecodeAny(c.DecodeFunc, kvs.Values(), result); err != nil {
			return "", nil, err
		}
	}

	return nextKey, kvs, nil
}

func (c *Cache) ScanRangeCallbackFn(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (string, int64, error) {
	nextKey, kvs, err := rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, 0, err
	}

	var read int64 = 0
	for _, kv := range kvs {
		if err != nil { // 出错的下一个key
			return kv.Key, read, err
		}
		read++

		if err = callback(kv); err != nil {
			//遇到错误时, continue, 根据上面err的判断, 方法会return错误后下一个key, 也就是nextKey
			continue
		}
	}

	return nextKey, read, err
}

func (c *Cache) ScanPrefixFn(keyPrefix string, result any, rangeFunc RangeFunc) (utils.KVs, error) {
	kvs := utils.KVs{}

	var keyStart = keyPrefix
	var keyEnd = utils.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs utils.KVs
	for {
		nextKey, _kvs, err = rangeFunc(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return nil, err
		}
		kvs = kvs.Add(_kvs)
		if nextKey == "" {
			break
		}
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err := text_utils.ListDecodeAny(c.DecodeFunc, kvs.Values(), result); err != nil {
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Cache) ScanPrefixCallbackFn(keyPrefix string, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (int64, error) {
	var keyStart = keyPrefix
	var keyEnd = utils.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var err error
	var _kvs utils.KVs
	var read int64
	for {
		nextKey, _kvs, err = rangeFunc(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return read, err
		}

		for _, kv := range _kvs {
			read++
			if err = callback(kv); err != nil {
				return read, err
			}
		}

		if nextKey == "" {
			return read, nil
		}
	}
}

func (c *Cache) GetDecodeFunc() text_utils.DecoderFunc {
	return c.DecodeFunc
}

func (c *Cache) GetEncodeFunc() text_utils.EncoderFunc {
	return c.EncodeFunc
}
