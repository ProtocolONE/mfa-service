FROM golang:1.11.2-alpine AS builder
MAINTAINER Vadim Sabirov <vadim.sabirov@protocol.one>

RUN apk add bash ca-certificates git

WORKDIR /application

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -o $GOPATH/bin/mfa-service .

ENTRYPOINT $GOPATH/bin/mfa-service
