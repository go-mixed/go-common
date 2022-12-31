package redis

import (
	"github.com/go-redis/redis/v9"
	"time"
)

type RedisOptions struct {
	// 1 host for single client/sentinel client
	// multi hosts for cluster client/fail-over client
	Addresses []string `yaml:"addrs" validate:"required,dive,hostname_port"`

	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	SentinelUsername string `yaml:"sentinel_username"`
	SentinelPassword string `yaml:"sentinel_password"`

	// for fail-over client only
	MasterName string `yaml:"master_name"`

	// for single/fail-over client
	DB         int  `yaml:"db" validate:"lte=16"`
	MaxRetries int  `yaml:"max_retries"`
	ReadOnly   bool `yaml:"read_only"`

	PoolSize     int           `yaml:"pool_size"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	PoolTimeout  time.Duration `yaml:"pool_timeout"`

	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`

	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `yaml:"max_connection_age"`

	IsPika bool `yaml:"is_pika"`
}

func (o *RedisOptions) ToRedisUniversalOptions() *redis.UniversalOptions {
	return &redis.UniversalOptions{
		Addrs:            o.Addresses,
		Username:         o.Username,
		Password:         o.Password,
		SentinelUsername: o.SentinelUsername,
		SentinelPassword: o.SentinelPassword,

		DB:         o.DB,
		MaxRetries: o.MaxRetries,

		DialTimeout:  o.ConnectTimeout,
		ReadTimeout:  o.ReadTimeout,
		WriteTimeout: o.WriteTimeout,

		PoolSize:    o.PoolSize,
		PoolTimeout: o.PoolTimeout,

		ConnMaxIdleTime: o.ConnMaxIdleTime,
		ConnMaxLifetime: o.ConnMaxLifetime,
		MaxIdleConns:    o.MaxIdleConns,
	}
}

func DefaultRedisOptions() *RedisOptions {
	return &RedisOptions{
		//Addresses:  []string{"127.0.0.1:6379"},
		DB:         0,
		MaxRetries: -1,
		ReadOnly:   false,

		PoolSize:     10,
		PoolTimeout:  30_000,
		MinIdleConns: 5,
		MaxIdleConns: 10,

		ConnectTimeout: 10_000,
		ReadTimeout:    30_000,
		WriteTimeout:   30_000,

		ConnMaxIdleTime: time.Minute,
		ConnMaxLifetime: time.Minute * 10,

		IsPika: false,
	}
}
