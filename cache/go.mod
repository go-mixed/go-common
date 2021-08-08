module go-common-cache

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.2
	github.com/patrickmn/go-cache v2.1.0+incompatible
	go-common v0.0.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sys v0.0.0-20210806184541-e5e7981a1069 // indirect
	google.golang.org/genproto v0.0.0-20210805201207-89edb61ffb67 // indirect
	google.golang.org/grpc v1.39.1 // indirect
)

replace go-common => ../
