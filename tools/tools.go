//go:build tools
// +build tools

// package tool pins a number of Go dependencies that we use. Go builds really fast,
// so wherever we can, we just build from source rather than pulling in third party binaries.
package main

//noinspection ALL
import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/fullstorydev/grpcurl/cmd/grpcurl"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/spf13/cobra-cli"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
