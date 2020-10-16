#!/bin/bash
set -e

go build -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
go build -o bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
go build -o bin/buf github.com/bufbuild/buf/cmd/buf
