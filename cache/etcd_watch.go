package cache

import (
	"context"
	"fmt"
	"go-common/utils"
	"go-common/utils/core"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
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

func (w *EtcdWatch) DumpAndWatch(ctx context.Context, keyPrefix string, fromRevision int64, handle EtcdHandle) (int64, error) {
	var revision int64
	var err error
	if revision, err = w.Dump(ctx, keyPrefix, fromRevision, -1, handle); err != nil {
		return revision, fmt.Errorf("dump cc from etcd error: %w", err)
	}

	if revision, err = w.Watch(ctx, keyPrefix, revision+1, handle); err != nil {
		return revision, fmt.Errorf("watch cc from etcd error: %w", err)
	}

	return revision, nil
}

func (w *EtcdWatch) Dump(ctx context.Context, keyPrefix string, fromRevision int64, toRevision int64, handler EtcdHandle) (int64, error) {
	if toRevision <= 0 {
		toRevision = w.etcd.LastRevisionByPrefix(keyPrefix)
	}
	var revision = fromRevision

	w.logger.Infof("start dump from etcd with key: \"%s\", revision: %d~%d", keyPrefix, fromRevision, toRevision)

	for {
		w.logger.Infof("dump revision %d~%d from etcd with key: \"%s\"", revision, core.If(toRevision > revision+20, revision+20, toRevision), keyPrefix)
		// 取出minRevision ~ maxRevision的kv(一个k只会出现1次), 按照revision 正序排序
		response, err := w.etcd.WithContext(ctx).PrefixResponseWithRev(
			keyPrefix,
			revision,
			toRevision,
			clientv3.WithLimit(20), // 每次取20个 避免阻塞太久
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
