module go-common-web

go 1.16

require (
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.3
	github.com/go-playground/validator/v10 v10.8.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/ugorji/go v1.2.6 // indirect
	go-common v0.0.0
	go.uber.org/zap v1.18.1
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go-common => ../
