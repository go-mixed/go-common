package etcd

import (
	"context"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gopkg.in/go-mixed/go-common.v1/utils"
	"gopkg.in/go-mixed/go-common.v1/utils/core"
)

type EtcdEventType int8

const (
	EtcdCreate EtcdEventType = iota + 1
	EtcdUpdate
	EtcdDelete
)

func (e EtcdEventType) String() string {
	switch e {
	case EtcdCreate:
		return "create"
	case EtcdUpdate:
		return "update"
	case EtcdDelete:
		return "delete"
	}
	return ""
}

type EtcdHandle interface {
	Handle(ctx context.Context, eventType EtcdEventType, preKv *mvccpb.KeyValue, kv *mvccpb.KeyValue) error
}

type EtcdHandleFn func(ctx context.Context, eventType EtcdEventType, preKv *mvccpb.KeyValue, kv *mvccpb.KeyValue) error

func (fn EtcdHandleFn) Handle(ctx context.Context, eventType EtcdEventType, preKv *mvccpb.KeyValue, kv *mvccpb.KeyValue) error {
	return fn(ctx, eventType, preKv, kv)
}

type EtcdWatch struct {
	etcd   *Etcd
	logger utils.ILogger
}

func NewEtcdWatch(etcd *Etcd, logger utils.ILogger) *EtcdWatch {
	return &EtcdWatch{
		etcd:   etcd,
		logger: logger,
	}
}

// DumpAndWatch 会导出fromRevision~到当前revision中符合keyPrefix要求的kv，然后持续watch
//
// 如果Ctx被cancel，或者遇到报错，返回函数内最后获取到的revision，和错误
func (w *EtcdWatch) DumpAndWatch(ctx context.Context, keyPrefix string, fromRevision int64, handle EtcdHandle) (int64, error) {
	var revision int64
	var err error
	if revision, err = w.Dump(ctx, keyPrefix, fromRevision, -1, handle); err != nil {
		return revision, errors.Errorf("dump cc from etcd error: %w", err)
	}

	if revision, err = w.Watch(ctx, keyPrefix, revision+1, handle); err != nil {
		return revision, errors.Errorf("watch cc from etcd error: %w", err)
	}

	return revision, nil
}

// Dump 按fromRevision~toRevision 导出所有符合keyPrefix的kv
//
// 注意：会无视compactRevision，会从fromRevision开始，只要key没被删除，使用此方法也能读出到小于compactRevision的key， 这点和watch有区别，
// 如果有必要，可以先尝试读取出compactRevision了，再传入本函数
func (w *EtcdWatch) Dump(ctx context.Context, keyPrefix string, fromRevision int64, toRevision int64, handler EtcdHandle) (int64, error) {
	if toRevision <= 0 {
		toRevision = w.etcd.LastRevisionByPrefix(keyPrefix)
	}
	var revision = fromRevision

	w.logger.Infof("start dump from etcd with key: \"%s\", revision: %d~%d", keyPrefix, fromRevision, toRevision)

	const c = 20
	for {
		w.logger.Infof("dump revision %d~%d from etcd with key: \"%s\"", revision, core.If(toRevision > revision+c, revision+c, toRevision), keyPrefix)
		// 取出minRevision ~ maxRevision的kv(一个k只会出现1次), 按照revision 正序排序
		response, err := w.etcd.WithContext(ctx).PrefixResponseWithRev(
			keyPrefix,
			revision,
			toRevision,
			clientv3.WithLimit(c), // 每次取20个 避免阻塞太久
		)

		if err != nil {
			return revision, err
		}

		if len(response.Kvs) <= 0 { // 无内容
			break
		}

		for _, kv := range response.Kvs {
			if core.IsContextDone(ctx) {
				w.logger.Infof("stop dump from etcd with key: \"%s\" revision: %d~%d", keyPrefix, fromRevision, revision)
				return revision, nil
			}
			revision = kv.ModRevision
			if err = handler.Handle(ctx, EtcdCreate, nil, kv); err != nil {
				return revision, err
			}
		}

		revision++ // 累加1, 为下一轮dump准备
	}

	w.logger.Infof("complete to dump etcd with key: \"%s\", revision: %d~%d", keyPrefix, fromRevision, revision)
	return revision, nil
}

// Watch 从fromRevision开始监听符合keyPrefix要求的kv
//
// 注意：当fromRevision < compactRevision时，会从compactRevision开始读取
func (w *EtcdWatch) Watch(ctx context.Context, keyPrefix string, fromRevision int64, handler EtcdHandle) (int64, error) {
	revision := fromRevision

	scopeCtx, cancel := context.WithCancel(ctx)
	defer cancel() // 在退出时必须被调用, 防止Watch的ch泄露

	for {
		if core.IsContextDone(scopeCtx) {
			break
		}

		w.logger.Infof("start watch etcd with key: \"%s\", revision >= %d", keyPrefix, revision)

		for event := range w.etcd.WatchWithContext(scopeCtx, keyPrefix, revision) {
			revision = event.Kv.ModRevision

			if err := handler.Handle(ctx, parseEtcdEventType(event), event.PrevKv, event.Kv); err != nil {
				return revision, err
			}
		}

		// 只有当revision有增加时, 才累加1, 也就是至少有一次for循环
		if revision != fromRevision {
			revision++ // 累加1, 为下一轮watch准备
		}
	}

	w.logger.Infof("complete to watch etcd with key: \"%s\", revision: %d~%d", keyPrefix, fromRevision, revision)
	return revision, nil

}

func parseEtcdEventType(event *clientv3.Event) EtcdEventType {
	if event.Type == mvccpb.DELETE {
		return EtcdDelete
	} else if event.IsCreate() {
		return EtcdCreate
	} else {
		return EtcdUpdate
	}
}
