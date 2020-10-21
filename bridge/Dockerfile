# syntax=docker/dockerfile:experimental
FROM golang:1.15.3

WORKDIR /app

ADD go.mod .
ADD go.sum .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

ADD . .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -gcflags="all=-N -l" -mod=readonly -o /guardiand github.com/certusone/wormhole/bridge/cmd/guardiand

ENTRYPOINT /guardiand
