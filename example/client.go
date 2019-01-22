package main

import (
	"context"
	"fmt"
	"github.com/micro/go-micro"
	"p1mfa/pkg"
	"p1mfa/pkg/proto"
)

func main() {
	// create a new service
	service := micro.NewService()

	// parse command line flags
	service.Init()

	// Create new greeter client
	client := proto.NewMfaService(mfa.ServiceName, service.Client())

	// Call it
	rsp, err := client.Create(context.TODO(), &proto.MfaCreateDataRequest{AppName: "Dummy", UserID: "12312312312313", Email: "test@test.com", QrSize: 300})
	if err != nil {
		fmt.Println(err)
	}
	// Print response
	fmt.Printf("%+v\n", rsp)

	// Call it
	rsp2, err2 := client.Check(context.TODO(), &proto.MfaCheckDataRequest{Code: "dummy", UserID: "12312312312313"})
	if err2 != nil {
		fmt.Println(err2)
	}
	// Print response
	fmt.Printf("%+v\n", rsp2)
}
