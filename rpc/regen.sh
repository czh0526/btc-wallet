#!/bin/sh

protoc -I. api.proto --go_out=. --go-grpc_out=./