# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM --platform=linux/arm64 docker.io/golang:1.21.9-bullseye@sha256:311468bffa9fa4747a334b94e6ce3681b564126d653675a6adc46698b2b88d35 AS build
# libwasmvm.so is not compatible with arm

FROM dev AS build

WORKDIR /app

COPY node/tools tools

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd tools/ && go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

COPY node node
COPY sdk sdk
COPY wormchain wormchain

ARG GO_BUILD_ARGS=-race

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  export CGO_ENABLED=1 && \
  cd node && \
  go build ${GO_BUILD_ARGS} -gcflags="all=-N -l" --ldflags '-extldflags "-Wl,--allow-multiple-definition" -X "github.com/certusone/wormhole/node/cmd/guardiand.Build=dev"' -mod=readonly -o /guardiand github.com/certusone/wormhole/node && \
  go get github.com/CosmWasm/wasmvm@v1.1.1 && \
  cp /go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.1.1/internal/api/libwasmvm.aarch64.so /usr/lib/

ENTRYPOINT ["/bin/bash"]