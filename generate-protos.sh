#!/bin/bash

(
  cd tools/
  go build -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
  go build -o bin/protoc-gen-go-grpc google.golang.org/grpc/cmd/protoc-gen-go-grpc
)

tools/bin/buf protoc \
  --plugin tools/bin/protoc-gen-go \
  --go_out=bridge/pkg/ proto/**/**/**

tools/bin/buf protoc \
   --plugin tools/bin/protoc-gen-go-grpc \
   --go-grpc_out=bridge/pkg/ proto/**/**/**
