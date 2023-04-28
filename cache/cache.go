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

	decoderFunc textUtils.DecoderFunc
	encoderFunc textUtils.EncoderFunc
}

func (c *Cache) L2() utils.IMemKV {
	return c.L2Cache
}

type RangeFunc func(keyStart, keyEnd string, keyPrefix string, limit int64) (nextKey string, kvs utils.KVs, err error)

func (c *Cache) SetEncoderFunc(encodeFunc textUtils.EncoderFunc) *Cache {
	c.encoderFunc = encodeFunc
	return c
}

func (c *Cache) EncoderFunc(v any) ([]byte, error) {
	return c.encoderFunc(v)
}

func (c *Cache) SetDecoderFunc(decodeFunc textUtils.DecoderFunc) *Cache {
	c.decoderFunc = decodeFunc
	return c
}

func (c *Cache) DecoderFunc(buf []byte, actual any) error {
	return c.decoderFunc(buf, actual)
}

// ScanRangeFn 遍历指定条件的数据，并导出到actual
//
//	keyStart、keyEnd为空表示从头遍历或遍历到结尾；keyPrefix为空表示前缀不限；limit为-1表示不限制数量
func (c *Cache) ScanRangeFn(keyStart, keyEnd string, keyPrefix string, limit int64, result any, rangeFunc RangeFunc) (nextKey string, kvs utils.KVs, err error) {
	nextKey, kvs, err = rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, kvs, err
	}

	if !core.IsInterfaceNil(result) && len(kvs) > 0 {
		if err = textUtils.ListDecodeAny(c.decoderFunc, kvs.Values(), result); err != nil {
			return "", nil, err
		}
	}

	return nextKey, kvs, nil
}

// ScanRangeCallbackFn 遍历指定条件的数据，每条数据callback，返回错误则跳出遍历
//
//	keyStart、keyEnd为空表示从头遍历或遍历到结尾；keyPrefix为空表示前缀不限；limit为-1表示不限制数量
func (c *Cache) ScanRangeCallbackFn(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (nextKey string, count int64, err error) {
	var kvs utils.KVs
	nextKey, kvs, err = rangeFunc(keyStart, keyEnd, keyPrefix, limit)
	if err != nil {
		return nextKey, 0, err
	}

	for _, kv := range kvs {
		if err != nil { // 出错的下一个key
			return kv.Key, count, err
		}
		count++

		if err = callback(kv); err != nil {
			//遇到错误时, continue, 根据上面err的判断, 方法会return错误后下一个key, 也就是nextKey
			continue
		}
	}

	return nextKey, count, err
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
		if err = textUtils.ListDecodeAny(c.decoderFunc, kvs.Values(), result); err != nil {
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Cache) ScanPrefixCallbackFn(keyPrefix string, callback func(kv *utils.KV) error, rangeFunc RangeFunc) (count int64, err error) {
	var keyStart = keyPrefix
	var keyEnd = utils.GetPrefixRangeEnd(keyPrefix)
	var nextKey = keyStart
	var _kvs utils.KVs
	for {
		nextKey, _kvs, err = rangeFunc(nextKey, keyEnd, keyPrefix, 10)
		if err != nil {
			return count, err
		}

		for _, kv := range _kvs {
			count++
			if err = callback(kv); err != nil {
				return count, err
			}
		}

		if nextKey == "" {
			return count, nil
		}
	}
}
