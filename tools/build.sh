#!/usr/bin/env bash
set -e

go build -mod=readonly -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
go build -mod=readonly -o bin/protoc-gen-grpc-gateway github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go build -mod=readonly -o bin/protoc-gen-openapiv2 github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2
go build -mod=readonly -o bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
go build -mod=readonly -o bin/buf github.com/bufbuild/buf/cmd/buf
go build -mod=readonly -o bin/cobra-cli github.com/spf13/cobra-cli
go build -mod=readonly -o bin/grpcurl github.com/fullstorydev/grpcurl/cmd/grpcurl
