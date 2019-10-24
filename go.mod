module github.com/ProtocolONE/mfa-service

require (
	github.com/InVisionApp/go-health v2.1.0+incompatible
	github.com/InVisionApp/go-logger v1.0.1 // indirect
	github.com/ProtocolONE/go-micro-plugins v0.2.0
	github.com/boombuler/barcode v1.0.0 // indirect
	github.com/go-redis/redis v6.15.1+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/micro/go-micro v1.8.0
	github.com/micro/go-plugins v1.2.0
	github.com/pquerna/otp v1.1.0
	github.com/prometheus/client_golang v1.0.0
	github.com/stretchr/testify v1.3.0
	go.uber.org/zap v1.10.0
)

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.5.1
