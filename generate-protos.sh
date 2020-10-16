#!/bin/bash

(
  cd tools/
  ./build.sh
)

tools/bin/buf protoc \
  --plugin tools/bin/protoc-gen-go \
  --go_out=bridge/pkg/ proto/**/**/**

tools/bin/buf protoc \
   --plugin tools/bin/protoc-gen-go-grpc \
   --go-grpc_out=bridge/pkg/ proto/**/**/**
