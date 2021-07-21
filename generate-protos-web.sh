#!/usr/bin/env bash

(
  cd tools/
  npm ci
)

mkdir -p explorer/src/proto

tools/bin/buf protoc \
  -Iproto \
  -Ithird_party/googleapis \
  --plugin tools/node_modules/.bin/protoc-gen-ts_proto \
  --ts_proto_opt=esModuleInterop=true \
  --ts_proto_opt=env=browser \
  --ts_proto_opt=forceLong=string \
  --ts_proto_opt=outputClientImpl=grpc-web \
  --ts_proto_out=explorer/src/proto/ proto/**/**/**
