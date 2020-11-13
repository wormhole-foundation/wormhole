# syntax=docker/dockerfile:experimental
FROM golang:1.15.3

WORKDIR /app

ADD tools tools

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd tools/ && go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

ADD . .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -gcflags="all=-N -l" -o /guardiand github.com/certusone/wormhole/bridge

ENTRYPOINT /guardiand
