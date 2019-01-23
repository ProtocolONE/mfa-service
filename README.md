mfa-service
=============

[![Build Status](https://travis-ci.org/ProtocolONE/mfa-service.svg?branch=master)](https://travis-ci.org/ProtocolONE/mfa-service) [![codecov](https://codecov.io/gh/ProtocolONE/mfa-service/branch/master/graph/badge.svg)](https://codecov.io/gh/ProtocolONE/mfa-service)


# Prerequisites
Requires a [go installation](https://golang.org/dl/).

## Running the service
This service works as [Go-Micro](https://github.com/micro/go-micro) microservice. You may want to 
setup your own registry with `MICRO_REGISTRY`/`MICRO_REGISTRY_ADDRESS` or use other go-micro flags.  

Download it 

`go get github.com/ProtocolONE/mfa-service`

If you need it uou can rebuild proto file with protoc 
```
protoc --proto_path=. --micro_out=. --go_out=. mfa.proto
```

By default service will be executed with declared by `MICRO_REGISTRY` registry and GRPC as a transport.

## Using Docker
The docker file in this project used to launch mfa-service in Protocol One environment. You may change it in any
way you need it.

# Using the service

Once the service is running you can use go-micro to make requests

```go
package main

import (
    "context"
    "fmt"
    "github.com/micro/go-micro"
    "github.com/ProtocolONE/mfa-service/pkg"
    "github.com/ProtocolONE/mfa-service/pkg/proto"
)

func main() {
    // create a new service
    service := micro.NewService()

    // parse command line flags
    service.Init()

    // Create new greeter client
    client := proto.NewMfaService(mfa.ServiceName, service.Client())

    // Call it
    rsp, err := client.Create(
    	context.TODO(), 
    	&proto.MfaCreateDataRequest{
    		ProviderID: "12312312312313",
    		AppName:    "dummy", 
    		UserID:     "123132123123",
    		Email:      "test@test.com",
    		QrSize:     300,
    	})
    if err != nil {
        fmt.Println(err)
    }

    // Print response
    fmt.Println(rsp)
}
```