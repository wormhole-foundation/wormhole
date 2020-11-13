#!/bin/bash
set -e

go build -mod=readonly -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
go build -mod=readonly -o bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
go build -mod=readonly -o bin/buf github.com/bufbuild/buf/cmd/buf
go build -mod=readonly -o bin/cobra github.com/spf13/cobra/cobra

