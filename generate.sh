#!/bin/bash

(
  cd tools/
  go build -o bin/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go
  go build -o bin/buf github.com/bufbuild/buf/cmd/buf
)

tools/bin/buf protoc --go_out=bridge/pkg/ proto/**/**/** --plugin tools/bin/protoc-gen-go
