#!/bin/bash

(
  cd tools/
  go build -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
)

tools/bin/buf protoc --go_out=bridge/pkg/ proto/**/**/** --plugin tools/bin/protoc-gen-go
