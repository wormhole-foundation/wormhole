#!/bin/bash

(
  cd tools/
  ./build.sh
)

(
  cd third_party/
  [[ ! -d googleapis ]] && git clone https://github.com/googleapis/googleapis
  cd googleapis
  git checkout 24fb9e5d1f37110bfa198189c34324aa3fdb0896
)

tools/bin/buf protoc \
  -Iproto \
  -Ithird_party/googleapis \
  --plugin tools/bin/protoc-gen-go \
  --go_out=bridge/pkg/ proto/**/**/**

tools/bin/buf protoc \
  -Iproto \
  -Ithird_party/googleapis \
  --plugin tools/bin/protoc-gen-go-grpc \
  --go-grpc_out=bridge/pkg/ proto/**/**/**
