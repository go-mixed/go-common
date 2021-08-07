module go-common-web

go 1.16

require (
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.2
	github.com/go-playground/validator/v10 v10.6.1 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/ugorji/go v1.2.6 // indirect
	go-common v0.0.0
	go-common-cache v0.0.0
	go.uber.org/zap v1.18.1
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace go-common => ../

replace go-common-cache => ../cache
