package redis

import (
	"github.com/go-redis/redis/v9"
	"time"
)

type RedisOptions struct {
	// 1个地址用于单机/哨兵客户端，多个地址用于集群/主从客户端
	// 1 host for single client/sentinel client
	// multi hosts for cluster client/fail-over client
	Addresses []string `yaml:"addrs" validate:"required,dive,hostname_port"`

	Username         string `yaml:"username"`
	Password         string `yaml:"password"`
	SentinelUsername string `yaml:"sentinel_username"`
	SentinelPassword string `yaml:"sentinel_password"`

	// 桶 0-15，只有单机/哨兵支持
	// databases 0-15, only single client/sentinel client
	DB int `yaml:"db" validate:"lte=16"`

	// 最大重试次数，默认值为3
	// the maximum number of retries, default value is 3
	MaxRetries int `yaml:"max_retries"`

	// 连接池最大连接数
	// the maximum number of connections in the pool
	PoolSize int `yaml:"pool_size"`
	// 客户端申请连接的最大等待时间（当所有连接都在忙碌是才会等待），超过此时间则返回错误
	// the timeout of client await for a connection from the pool
	PoolTimeout time.Duration `yaml:"pool_timeout"`

	// 最大空闲连接数
	// the maximum number of idle connections in the pool
	MaxIdleConns int `yaml:"max_idle_conns"`
	// 最小空闲连接数，至少会保留这么多的空闲连接
	// the minimum number of idle connections in the pool, at least keep this number of idle connections
	MinIdleConns int `yaml:"min_idle_conns"`

	// 连接超时时间，默认值为5秒，在Redis配置命名为DialTimeout
	// the timeout of dialing a connection
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	// 读超时时间，默认值为3秒
	// the timeout of reading a connection, default value is 3 seconds
	ReadTimeout time.Duration `yaml:"read_timeout"`
	// 写超时时间，默认值为3秒
	// the timeout of writing a connection, default value is 3 seconds
	WriteTimeout time.Duration `yaml:"write_timeout"`

	// 连接最大空闲时间，默认值为30分钟，超过此时间则关闭连接
	// the maximum idle time of a connection, default value is 30 minutes, close the connection if it exceeds this time
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	// 连接最大存活时间，默认值为0，不会关闭连接
	// the maximum life-time of a connection, default value is 0, never close the connection
	ConnMaxLifetime time.Duration `yaml:"max_connection_age"`

	// 是否是Pika的服务端，会支持一些Pika的特性，比如：pkscanrange
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

		PoolSize:    10,
		PoolTimeout: 30_000,

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
