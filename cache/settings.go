package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"go-common/utils/time"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"time"
)

type EtcdConfig struct {
	Endpoints            []string                       `json:"endpoints" yaml:"endpoints" validate:"required"`
	AutoSyncInterval     time_utils.MillisecondDuration `json:"auto_sync_interval" yaml:"auto_sync_interval"`
	DialTimeout          time_utils.MillisecondDuration `json:"dial_timeout" yaml:"dial_timeout"`
	DialKeepAliveTime    time_utils.MillisecondDuration `json:"dial_keep_alive_time" yaml:"dial_keep_alive_time"`
	DialKeepAliveTimeout time_utils.MillisecondDuration `json:"dial_keep_alive_timeout" yaml:"dial_keep_alive_timeout"`
	Username             string                         `json:"username" yaml:"username"`
	Password             string                         `json:"password" yaml:"password"`
}

type RedisOptions struct {
	// 1 host for single client/sentinel client
	// multi hosts for cluster client/fail-over client
	Addresses []string `json:"addrs" yaml:"addrs" validate:"required,dive,hostname_port"`

	Username         string `json:"username" yaml:"username"`
	Password         string `json:"password" yaml:"password"`
	SentinelPassword string `json:"sentinel_password" yaml:"sentinel_password"`

	// for fail-over client only
	MasterName string `json:"master_name" yaml:"master_name"`

	// for single/fail-over client
	DB         int  `json:"db" yaml:"db" validate:"lte=16"`
	MaxRetries int  `json:"max_retries" yaml:"max_retries"`
	ReadOnly   bool `json:"read_only" yaml:"read_only"`

	PoolSize    int                            `json:"pool_size" yaml:"pool_size"`
	PoolTimeout time_utils.MillisecondDuration `json:"pool_timeout" yaml:"pool_timeout"`

	ConnectTimeout time_utils.MillisecondDuration `json:"connect_timeout" yaml:"connect_timeout"`
	ReadTimeout    time_utils.MillisecondDuration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout   time_utils.MillisecondDuration `json:"write_timeout" yaml:"write_timeout"`
	MaxConnAge     time_utils.MillisecondDuration `json:"max_connection_age" yaml:"max_connection_age"`
}

func (o *RedisOptions) ToRedisUniversalOptions() *redis.UniversalOptions {
	return &redis.UniversalOptions{
		Addrs:              o.Addresses,
		Username:           o.Username,
		Password:           o.Password,
		SentinelPassword:   o.SentinelPassword,
		DB:                 o.DB,
		MaxRetries:         o.MaxRetries,
		DialTimeout:        o.ConnectTimeout.ToDuration(),
		ReadTimeout:        o.ReadTimeout.ToDuration(),
		WriteTimeout:       o.WriteTimeout.ToDuration(),
		PoolSize:           o.PoolSize,
		MaxConnAge:         o.MaxConnAge.ToDuration(),
		PoolTimeout:        o.PoolTimeout.ToDuration(),
		IdleTimeout:        time.Minute,
		IdleCheckFrequency: 100 * time.Millisecond,
	}
}

func (c EtcdConfig) ToEtcdConfig(logger *zap.Logger) *clientv3.Config {
	return &clientv3.Config{
		Endpoints:            c.Endpoints,
		AutoSyncInterval:     c.AutoSyncInterval.ToDuration(),
		DialTimeout:          c.DialTimeout.ToDuration(),
		DialKeepAliveTime:    c.DialKeepAliveTime.ToDuration(),
		DialKeepAliveTimeout: c.DialKeepAliveTimeout.ToDuration(),

		Logger:   logger,
		Username: c.Username,
		Password: c.Password,

		Context: context.Background(),
	}
}

func DefaultRedisOptions() *RedisOptions {
	return &RedisOptions{
		//Addresses:  []string{"127.0.0.1:6379"},
		DB:         0,
		MaxRetries: -1,
		ReadOnly:   false,

		PoolSize:    10,
		PoolTimeout: 30_000,

		ConnectTimeout: 10_000,
		ReadTimeout:    30_000,
		WriteTimeout:   30_000,
		MaxConnAge:     0,
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
