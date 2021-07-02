#!/usr/bin/env bash

(
  cd tools/
  ./build.sh
  npm ci
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
  --go_opt=module=github.com/certusone/wormhole/bridge/pkg \
  --go_out=bridge/pkg/ proto/**/**/**

tools/bin/buf protoc \
  -Iproto \
  -Ithird_party/googleapis \
  --plugin tools/bin/protoc-gen-go-grpc \
  --go-grpc_opt=module=github.com/certusone/wormhole/bridge/pkg \
  --go-grpc_out=bridge/pkg/ proto/**/**/**

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
