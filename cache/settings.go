package cache

import (
	"github.com/go-redis/redis/v8"
	"go-common/utils"
	"go-common/utils/time"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type EtcdConfig struct {
	Endpoints            []string                       `json:"endpoints"`
	AutoSyncInterval     time_utils.MillisecondDuration `json:"auto_sync_interval"`
	DialTimeout          time_utils.MillisecondDuration `json:"dial_timeout"`
	DialKeepAliveTime    time_utils.MillisecondDuration `json:"dial_keep_alive_time"`
	DialKeepAliveTimeout time_utils.MillisecondDuration `json:"dial_keep_alive_timeout"`
	Username             string                         `json:"username"`
	Password             string                         `json:"password"`
}

type RedisOptions struct {
	// 1 host for single client/sentinel client
	// multi hosts for cluster client/fail-over client
	Addresses []string `json:"addrs"`

	Username         string `json:"username"`
	Password         string `json:"password"`
	SentinelPassword string `json:"sentinel_password"`

	// for fail-over client only
	MasterName string `json:"master_name"`

	// for single/fail-over client
	DB         int  `json:"db"`
	MaxRetries int  `json:"max_retries"`
	ReadOnly   bool `json:"read_only"`

	PoolSize    int                            `json:"pool_size"`
	PoolTimeout time_utils.MillisecondDuration `json:"pool_timeout"`

	ConnectTimeout time_utils.MillisecondDuration `json:"connect_timeout"`
	ReadTimeout    time_utils.MillisecondDuration `json:"read_timeout"`
	WriteTimeout   time_utils.MillisecondDuration `json:"write_timeout"`
	MaxConnAge     time_utils.MillisecondDuration `json:"max_connection_age"`
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

func (c EtcdConfig) ToEtcdConfig() *clientv3.Config {
	return &clientv3.Config{
		Endpoints:            c.Endpoints,
		AutoSyncInterval:     c.AutoSyncInterval.ToDuration(),
		DialTimeout:          c.DialTimeout.ToDuration(),
		DialKeepAliveTime:    c.DialKeepAliveTime.ToDuration(),
		DialKeepAliveTimeout: c.DialKeepAliveTimeout.ToDuration(),

		Logger:   utils.GetLogger(),
		Username: c.Username,
		Password: c.Password,
	}
}

func DefaultRedisOptions() *RedisOptions {
	return &RedisOptions{
		Addresses:  []string{"127.0.0.1:6379"},
		DB:         0,
		MaxRetries: -1,
		ReadOnly:   false,

		PoolSize:    10,
		PoolTimeout: 30000,

		ConnectTimeout: 10000,
		ReadTimeout:    30000,
		WriteTimeout:   30000,
		MaxConnAge:     0,
	}
}

func DefaultEtcdConfig() *EtcdConfig {
	return &EtcdConfig{
		Endpoints:            []string{"127.0.0.1:2379"},
		AutoSyncInterval:     10_000,
		DialTimeout:          5_000,
		DialKeepAliveTime:    100_000,
		DialKeepAliveTimeout: 10_000,
		Username:             "",
		Password:             "",
	}
}
