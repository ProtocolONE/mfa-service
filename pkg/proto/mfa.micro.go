// Code generated by protoc-gen-micro. DO NOT EDIT.
// source: mfa.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	mfa.proto

It has these top-level messages:
	MfaCreateDataRequest
	MfaCreateDataResponse
	MfaCheckDataRequest
	MfaCheckDataResponse
	Error
*/
package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "context"
	client "github.com/micro/go-micro/client"
	server "github.com/micro/go-micro/server"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto1.ProtoPackageIsVersion2 // please upgrade the proto package

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ client.Option
var _ server.Option

// Client API for MfaService service

type MfaService interface {
	Create(ctx context.Context, in *MfaCreateDataRequest, opts ...client.CallOption) (*MfaCreateDataResponse, error)
	Check(ctx context.Context, in *MfaCheckDataRequest, opts ...client.CallOption) (*MfaCheckDataResponse, error)
}

type mfaService struct {
	c    client.Client
	name string
}

func NewMfaService(name string, c client.Client) MfaService {
	if c == nil {
		c = client.NewClient()
	}
	if len(name) == 0 {
		name = "proto"
	}
	return &mfaService{
		c:    c,
		name: name,
	}
}

func (c *mfaService) Create(ctx context.Context, in *MfaCreateDataRequest, opts ...client.CallOption) (*MfaCreateDataResponse, error) {
	req := c.c.NewRequest(c.name, "MfaService.Create", in)
	out := new(MfaCreateDataResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mfaService) Check(ctx context.Context, in *MfaCheckDataRequest, opts ...client.CallOption) (*MfaCheckDataResponse, error) {
	req := c.c.NewRequest(c.name, "MfaService.Check", in)
	out := new(MfaCheckDataResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for MfaService service

type MfaServiceHandler interface {
	Create(context.Context, *MfaCreateDataRequest, *MfaCreateDataResponse) error
	Check(context.Context, *MfaCheckDataRequest, *MfaCheckDataResponse) error
}

func RegisterMfaServiceHandler(s server.Server, hdlr MfaServiceHandler, opts ...server.HandlerOption) error {
	type mfaService interface {
		Create(ctx context.Context, in *MfaCreateDataRequest, out *MfaCreateDataResponse) error
		Check(ctx context.Context, in *MfaCheckDataRequest, out *MfaCheckDataResponse) error
	}
	type MfaService struct {
		mfaService
	}
	h := &mfaServiceHandler{hdlr}
	return s.Handle(s.NewHandler(&MfaService{h}, opts...))
}

type mfaServiceHandler struct {
	MfaServiceHandler
}

func (h *mfaServiceHandler) Create(ctx context.Context, in *MfaCreateDataRequest, out *MfaCreateDataResponse) error {
	return h.MfaServiceHandler.Create(ctx, in, out)
}

func (h *mfaServiceHandler) Check(ctx context.Context, in *MfaCheckDataRequest, out *MfaCheckDataResponse) error {
	return h.MfaServiceHandler.Check(ctx, in, out)
}
