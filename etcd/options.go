package etcd

import (
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"time"
)

type EtcdConfig struct {
	Endpoints            []string      `yaml:"endpoints" validate:"required"`
	AutoSyncInterval     time.Duration `yaml:"auto_sync_interval"`
	DialTimeout          time.Duration `yaml:"dial_timeout"`
	DialKeepAliveTime    time.Duration `yaml:"dial_keep_alive_time"`
	DialKeepAliveTimeout time.Duration `yaml:"dial_keep_alive_timeout"`
	Username             string        `yaml:"username"`
	Password             string        `yaml:"password"`
}

func (c EtcdConfig) ToEtcdConfig(logger *zap.Logger) *clientv3.Config {
	return &clientv3.Config{
		Endpoints:            c.Endpoints,
		AutoSyncInterval:     c.AutoSyncInterval,
		DialTimeout:          c.DialTimeout,
		DialKeepAliveTime:    c.DialKeepAliveTime,
		DialKeepAliveTimeout: c.DialKeepAliveTimeout,

		Logger:   logger,
		Username: c.Username,
		Password: c.Password,

		Context: context.Background(),
	}
}

func DefaultEtcdConfig() *EtcdConfig {
	return &EtcdConfig{
		//Endpoints:            []string{"127.0.0.1:2379"},
		AutoSyncInterval:     10_000,
		DialTimeout:          5_000,
		DialKeepAliveTime:    100_000,
		DialKeepAliveTimeout: 10_000,
		Username:             "",
		Password:             "",
	}
}
