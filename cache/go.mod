module go-common-cache

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	go-common v0.0.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
)

replace go-common => ../
