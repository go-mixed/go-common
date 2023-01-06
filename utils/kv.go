package utils

import (
	"time"
)

type KV struct {
	Key   string
	Value []byte
}

type KVs []*KV

func (s KVs) Append(k string, v []byte) KVs {
	return append(s, &KV{
		Key:   k,
		Value: v,
	})
}

func (s KVs) Add(_new KVs) KVs {
	return append(s, _new...)
}

func (s KVs) Keys() []string {
	var keys []string
	for _, kv := range s {
		keys = append(keys, kv.Key)
	}
	return keys
}

func (s KVs) Values() [][]byte {
	var values [][]byte
	for _, kv := range s {
		values = append(values, kv.Value)
	}
	return values
}

func NewKV(key string, value []byte) *KV {
	return &KV{Key: key, Value: value}
}

type KVBatchFunc func(client IKV) error

type IKV interface {
	// L2 得到本Cache的二级缓存对象
	L2() IMemKV
	// Get 查询key的值, 并尝试将其值导出到actual 如果无需导出, actual 传入nil
	Get(key string, actual any) ([]byte, error)
	// MGet 查询多个keys, 返回所有符合要求KV, 并尝试将数据导出到actual 如果无需导出, actual 传入nil
	//  例子:
	//  var result []User
	//  RedisGet(keys, &result)
	//  注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	MGet(keys []string, actual any) (KVs, error)

	// Keys keyPrefix为前缀 返回所有符合要求的keys
	//  注意: 遇到有太多的匹配项, 会阻塞运行
	Keys(keyPrefix string) ([]string, error)
	// Range 返回在keyStart（含）~keyEnd（含）中遍历符合keyPrefix要求的KV
	//  keyStart、keyEnd为空表示从头遍历或遍历到结尾；keyPrefix为空表示不限制前缀；limit为-1表示不限制数量（仅Pika）
	//  返回nextKey, kv列表, 错误
	Range(keyStart, keyEnd string, keyPrefix string, limit int64) (nextKey string, kvs KVs, err error)
	// ScanPrefix keyPrefix为前缀, 返回所有符合条件的KV, 并尝试将数据导出到actual 如果无需导出, actual 传入nil
	//  注意: 不要在keyPrefix中或结尾加入*
	// 例子:
	//  var result []User
	//  ScanPrefix("users/id/", &result)
	//  - 注意: result必须是slice, 如果kvs中有一个值无法转换, 都将返回错误, 所以这些keys一定要拥有相同的结构
	//  - 注意: 如果有太多的匹配项, 会阻塞运行. 对于大的量级, 尽量使用 ScanPrefixCallback
	ScanPrefix(keyPrefix string, actual any) (KVs, error)
	// ScanPrefixCallback 根据keyPrefix为前缀 查询出所有K/V 遍历调用callback
	//  如果callback返回nil, 会一直搜索直到再无匹配数据; 如果返回错误, 则立即停止搜索
	//  注意: 即使cache中有大量的匹配项, 也不会被阻塞
	ScanPrefixCallback(keyPrefix string, callback func(kv *KV) error) (count int64, err error)

	// ScanRange 在keyStart（含）~keyEnd（含）中遍历符合keyPrefix要求的KV, 并尝试将数据导出到actual 如果无需导出, actual 传入nil
	//   keyStart、keyEnd为空表示从头遍历或遍历到结尾；keyPrefix为空表示不限制前缀；limit为-1表示不限制数量（仅Pika）
	// 例子:
	//  var result []User
	//  // 从 "users/id/100" 开始, 取前缀为"users/id/"的100个数据
	//  ScanRange("users/id/100", "", "users/id/", 100, &result)
	//  // 比如取a~z的所有数据, 会包含 "a", "a1", "a2xxxxxx", "yyyyyy", "z"
	//  ScanRange("a", "z", "", 0, &result)
	//  注意: result必须要是slice, 并且只要有一个值无法转换, 都返回错误, 所以这些keys一定要拥有相同的结构
	ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, actual any) (nextKey string, kvs KVs, err error)
	// ScanRangeCallback 在keyStart（含）~keyEnd（含）中遍历符合keyPrefix要求KV，遍历调用callback，返回错误则停止遍历
	//  keyStart、keyEnd为空表示从头遍历或遍历到结尾；keyPrefix为空表示不限制前缀；limit为-1表示不限制数量（仅Pika）
	ScanRangeCallback(keyStart, keyEnd string, keyPrefix string, limit int64, callback func(kv *KV) error) (nextKey string, count int64, err error)

	// Set 写入KV
	Set(key string, val any, expiration time.Duration) error
	SetNoExpiration(key string, val any) error
	Del(key string) error

	// DecoderFunc 导出[]byte到actual
	DecoderFunc([]byte, any) error
	// EncoderFunc 将任意类型导出为 []byte
	EncoderFunc(any) ([]byte, error)

	// Batch 批量操作
	Batch(callback KVBatchFunc) error
}

type IMemKV interface {
	Get(key string, expire time.Duration, actual any) ([]byte, error)
	MGet(keys []string, expire time.Duration, actual any) (KVs, error)
	Keys(keyPrefix string, expire time.Duration) ([]string, error)
	Delete(keys ...string)

	ScanPrefix(keyPrefix string, expire time.Duration, actual any) (KVs, error)
}

// GetPrefixRangeEnd gets the range end of the prefix.
// 'Get(foo, WithPrefix())' is equal to 'Get(foo, WithRange(GetPrefixRangeEnd(foo))'.
func GetPrefixRangeEnd(prefix string) string {
	return string(getPrefix([]byte(prefix)))
}

var noPrefixEnd = []byte{0}

func getPrefix(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return end
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to WithFromKey policy
	return noPrefixEnd
}
