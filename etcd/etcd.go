package etcd

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/go-mixed/go-common.v1/cache.v1"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"strings"
	"time"
)

type Etcd struct {
	cache.Cache
	EtcdClient *clientv3.Client
}

var _ utils.IKV = (*Etcd)(nil)

func (c *Etcd) SetNoExpiration(key string, val any) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.EtcdClient.Put(c.Ctx, key, text_utils.ToString(val, true))
	if err != nil {
		return err
	}

	return nil
}

func (c *Etcd) Del(key string) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]Del %s, %0.6f", key, time.Since(now).Seconds())
	}()

	_, err := c.EtcdClient.Delete(c.Ctx, key)
	if err != nil {
		return err
	}

	return nil
}

func (c *Etcd) Set(key string, val any, expiration time.Duration) error {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]Set %s, %0.6f", key, time.Since(now).Seconds())
	}()

	ttl := int64(expiration.Seconds())
	if ttl <= 0 {
		return c.SetNoExpiration(key, val)
	}

	lease := clientv3.NewLease(c.EtcdClient)
	response, err := lease.Grant(c.Ctx, ttl)
	if err != nil {
		return err
	}

	_, err = c.EtcdClient.Put(c.Ctx, key, text_utils.ToString(val, true), clientv3.WithLease(response.ID))
	if err != nil {
		return err
	}

	return nil
}

func (c *Etcd) Get(key string, actual any) ([]byte, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Infof("[ETCD]Get %s, %0.6f", key, time.Since(now).Seconds())
	}()
	kv := clientv3.NewKV(c.EtcdClient)
	response, err := kv.Get(c.Ctx, key, clientv3.WithLimit(1))
	if err != nil {
		c.Logger.Debugf("[ETCD]error of key %s", key, err.Error())
		return nil, err
	} else if len(response.Kvs) == 0 {
		c.Logger.Debugf("[ETCD]key not exists: %s", key)
		return nil, nil
	} else if len(response.Kvs[0].Value) == 0 {
		c.Logger.Debugf("[ETCD]empty value of key %s", key)
		return nil, nil
	}

	var val = response.Kvs[0].Value
	if !core.IsInterfaceNil(actual) {
		if err := c.DecodeFunc(val, actual); err != nil {
			c.Logger.Errorf("[ETCD]unmarshal: %s of error: %s", val, err.Error())
			return val, err
		}
	}
	return val, nil
}

func (c *Etcd) MGet(keys []string, actual any) (utils.KVs, error) {
	kv := clientv3.NewKV(c.EtcdClient)

	kvs := utils.KVs{}
	for _, key := range keys {
		response, err := kv.Get(c.Ctx, key, clientv3.WithLimit(1))
		if err != nil {
			return nil, err
		} else if len(response.Kvs) == 0 {
			kvs = kvs.Append(key, nil)
		} else {
			kvs = kvs.Append(key, response.Kvs[0].Value)
		}
	}

	if !core.IsInterfaceNil(actual) && len(kvs) > 0 {
		if err := text_utils.ListDecodeAny(c.DecodeFunc, kvs.Values(), actual); err != nil {
			c.Logger.Errorf("[ETCD]unmarshal: %v of error: %s", kvs.Values(), err.Error())
			return nil, err
		}
	}

	return kvs, nil
}

func (c *Etcd) Keys(keyPrefix string) ([]string, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]Keys %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()
	kv := clientv3.NewKV(c.EtcdClient)
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

func (c *Etcd) Range(keyStart, keyEnd string, keyPrefix string, limit int64) (string, utils.KVs, error) {
	kv := clientv3.NewKV(c.EtcdClient)

	kvs := utils.KVs{}

	response, err := kv.Get(c.Ctx, keyStart, clientv3.WithFromKey(), clientv3.WithRange(keyEnd), clientv3.WithLimit(limit+1)) // 多取1个是为了返回最后一个为nextKey
	if err != nil {
		return "", nil, err
	} else if len(response.Kvs) == 0 {
		return "", nil, nil
	}
	var i int64
	count := core.If(limit < int64(len(response.Kvs)), limit, int64(len(response.Kvs)))
	for i = 0; i < count; i++ {
		key := string(response.Kvs[i].Key)
		if keyPrefix == "" || strings.HasPrefix(key, keyPrefix) {
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

func (c *Etcd) ScanPrefix(keyPrefix string, result any) (utils.KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]ScanPrefix %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	return c.ScanPrefixFn(keyPrefix, result, c.Range)
}

func (c *Etcd) ScanPrefixCallback(keyPrefix string, callback func(kv *utils.KV) error) (int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]ScanPrefixCallback %s, %0.6f", keyPrefix, time.Since(now).Seconds())
	}()

	return c.ScanPrefixCallbackFn(keyPrefix, callback, c.Range)
}

func (c *Etcd) ScanRange(keyStart, keyEnd string, keyPrefix string, limit int64, result any) (string, utils.KVs, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]ScanRange: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()
	return c.ScanRangeFn(keyStart, keyEnd, keyPrefix, limit, result, c.Range)
}

func (c *Etcd) ScanRangeCallback(keyStart string, keyEnd string, keyPrefix string, limit int64, callback func(kv *utils.KV) error) (string, int64, error) {
	var now = time.Now()
	defer func() {
		c.Logger.Debugf("[ETCD]ScanRangeCallback: keyStart: \"%s\", keyEnd: \"%s\", keyPrefix: \"%s\", limit: \"%d\", %0.6f", keyStart, keyEnd, keyPrefix, limit, time.Since(now).Seconds())
	}()

	return c.ScanRangeCallbackFn(keyStart, keyEnd, keyPrefix, limit, callback, c.Range)
}

func (c *Etcd) Close() error {
	return c.EtcdClient.Close()
}

//------ 自有的相关方法 ------

// LastRevision etcd的全局的最后版本号, 需要管理员权限
func (c *Etcd) LastRevision() int64 {
	return c.LastRevisionByPrefix("\x00")
}

// LastRevisionByPrefix 以prefix开头的最后版本号
func (c *Etcd) LastRevisionByPrefix(keyPrefix string) int64 {
	response, err := c.EtcdClient.Get(c.Ctx, keyPrefix, clientv3.WithLastRev()...)
	if err != nil {
		c.Logger.Warnf("get last revision with prefix \"%s\" error: %s", keyPrefix, err.Error())
		return -1
	}
	return response.Header.GetRevision()
}

// CompactRevisionByPrefix 获取压缩修订版, 请传递一个有权限的key前缀, 如果为空, 确保有管理员权限
// 如果有compact会立即返回数据, 如果没有compact, 则会在500ms之后返回0。
// 所以函数尽量用在有compact的情况下
func (c *Etcd) CompactRevisionByPrefix(keyPrefix string) int64 {
	if keyPrefix == "" {
		keyPrefix = "\x00"
	}

	ctx, cancel := context.WithTimeout(c.Ctx, 500*time.Millisecond)
	defer cancel()
	var compactRevision int64
	for ch := range c.EtcdClient.Watch(ctx, keyPrefix, clientv3.WithPrefix(), clientv3.WithRev(1)) {
		if ch.CompactRevision != 0 {
			compactRevision = ch.CompactRevision
		}
		break
	}

	return compactRevision
}

func (c *Etcd) GetResponse(key string, ops ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	kv := clientv3.NewKV(c.EtcdClient)
	return kv.Get(c.Ctx, key, ops...)
}

// PrefixResponse 得到前缀符合 keyPrefix 的所有值
func (c *Etcd) PrefixResponse(keyPrefix string, ops ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	ops = append(ops, clientv3.WithPrefix())
	return c.GetResponse(keyPrefix, ops...)
}

// PrefixResponseWithRev 得到指定版本范围内的 keyPrefix 列表 (一个key只会返回1次, value为最新的数据)
// 如果maxRev 不为0会返回指定minRev ~ maxRev版本范围的kv, 0表示minRev ~ +inf
// 数据依照ModifyRevision Asc排列, 所以当设置clientv3.WithLimit(N)时, 可以使用minRev参数来翻页 (即: 下一页的minRev = 返回结果的最后一条.ModRevision + 1)
func (c *Etcd) PrefixResponseWithRev(keyPrefix string, minRev, maxRev int64, ops ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if minRev < 0 {
		minRev = 0
	}
	if maxRev <= 0 {
		ops = append(ops, clientv3.WithMinModRev(minRev), clientv3.WithSort(clientv3.SortByModRevision, clientv3.SortAscend))
	} else {
		ops = append(ops, clientv3.WithMinModRev(minRev), clientv3.WithMaxModRev(maxRev), clientv3.WithSort(clientv3.SortByModRevision, clientv3.SortAscend))
	}

	return c.PrefixResponse(keyPrefix, ops...)
}

// WatchWithContext 监控key的变更, 并传入一个可以cancel的Context来控制watch是否停止，其它解释看 Watch
func (c *Etcd) WatchWithContext(ctx context.Context, keyPrefix string, minRev int64, opts ...clientv3.OpOption) <-chan *clientv3.Event {
	watcher := clientv3.NewWatcher(c.EtcdClient)
	outCh := make(chan *clientv3.Event)
	// 当不再需要watch（即非watch自己主动退出），必须关闭ctx
	innerCtx, innerCancel := context.WithCancel(ctx)
	go func() {
		defer innerCancel()
		defer close(outCh) // 总是会关闭此通道
	loop1:
		for {
			// 如果context已经被cancel, 则退出
			if core.IsContextDone(ctx) {
				c.Logger.Infof("[ETCD]watcher stop, prefix-key: \"%s\"", keyPrefix)
				return
			}

			var _opts []clientv3.OpOption
			if keyPrefix == "" {
				_opts = append(_opts, clientv3.WithFromKey())
			} else {
				_opts = append(_opts, clientv3.WithPrefix())
			}
			_opts = append(_opts, clientv3.WithRev(minRev), clientv3.WithPrevKV())
			_opts = append(_opts, opts...)
			// The context "ctx" MUST be canceled, as soon as watcher is no longer being used to release the associated resources.
			for response := range watcher.Watch(innerCtx, keyPrefix, _opts...) {
				if response.CompactRevision != 0 {
					c.Logger.Warnf("[ETCD]required revision has been compacted, key: \"%s\", compact revision: %d, required-revision: %d", keyPrefix, response.CompactRevision, minRev)
					minRev = response.CompactRevision
					// If revisions waiting to be sent over the watch are compacted,
					// then the watch will be canceled by the server,
					// the client will post a compacted error watch response, and the channel will close.
					continue loop1
				}
				if response.Canceled {
					c.Logger.Warnf("[ETCD]watcher is canceled, key: \"%s\", revision: %d, error: %v", keyPrefix, minRev, response.Err())
					return
				}

				for _, event := range response.Events {
					select {
					case outCh <- event: // 当通道未读取会一直阻塞
						minRev = event.Kv.ModRevision
					case <-ctx.Done(): // 监控ctx退出
						c.Logger.Infof("[ETCD]watcher stop in event loop, key: \"%s\"", keyPrefix)
						return
					}
				}
			}
			c.Logger.Infof("[ETCD]watcher close, key: \"%s\"", keyPrefix)
		}
	}()

	return outCh
}

// Watch 监视key的变更, 返回一个可以遍历所有变更的kv的通道. 如果keyPrefix为空, 则表示监视所有KEY, 如果minRev > 0 则从指定版本开始
// 方法返回ch, cancel := Watch(...) 使用cancel()可以强制终止watch
// cancel必须要被调用, 不然会内存泄露
func (c *Etcd) Watch(keyPrefix string, minRev int64, opts ...clientv3.OpOption) (<-chan *clientv3.Event, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Ctx)

	if minRev <= 0 {
		minRev = 1
	}

	ch := c.WatchWithContext(ctx, keyPrefix, minRev, opts...)
	return ch, func() {
		c.Logger.Infof("[ETCD]cancel watch: \"%s\" after revision: %d", keyPrefix, minRev)
		cancel()
	}
}

// RangeResponse 得到keyStart~keyEnd范围内，并且符合keyPrefix的数据, limit <= 0则表示不限制数量
// keyPrefix为空 表示无前缀要求
func (c *Etcd) RangeResponse(keyStart string, keyEnd string, keyPrefix string, limit int64, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if limit < 0 {
		limit = 0
	}

	opts = append(opts, clientv3.WithFromKey(), clientv3.WithRange(keyEnd), clientv3.WithLimit(limit))

	response, err := c.GetResponse(keyStart, opts...)
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

	return response, nil
}

func (c *Etcd) WithContext(ctx context.Context) *Etcd {
	newEtcd := *c
	newEtcd.Ctx = ctx
	return &newEtcd
}

// Lease 得到Lease的Response
func (c *Etcd) Lease(leaseID int64) (*clientv3.LeaseTimeToLiveResponse, error) {
	id := clientv3.LeaseID(leaseID)
	if id == clientv3.NoLease {
		return nil, errors.Errorf("it is not a valid lease id: %d", leaseID)
	}

	return c.EtcdClient.TimeToLive(c.Ctx, id)
}

// TimeToLive 得到某leaseID还剩余的TTL, 单位s
func (c *Etcd) TimeToLive(leaseID int64) (int64, error) {
	leaseResponse, err := c.Lease(leaseID)
	if err != nil {
		return -1, err
	}

	return leaseResponse.TTL, nil
}

func (c *Etcd) Batch(callback utils.KVBatchFunc) error {
	// Todo：Txn(context.TODO()).If(
	// Compare(Value(k1), ">", v1),
	// Compare(Version(k1), "=", 2)
	//).Then(
	// OpPut(k2,v2), OpPut(k3,v3)
	//).Else(
	// OpPut(k4,v4), OpPut(k5,v5)
	//).Commit()
	return callback(c)
}
