module go-common-cache

go 1.16

require (
	github.com/go-redis/redis/v8 v8.11.1
	github.com/lestrrat-go/strftime v1.0.5 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	go-common v0.0.0
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/v3 v3.5.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/zap v1.18.1 // indirect
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985 // indirect
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/genproto v0.0.0-20210729151513-df9385d47c1b // indirect
)

replace go-common => ../
