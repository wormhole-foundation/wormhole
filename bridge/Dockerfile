# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44
FROM golang:1.15.5

WORKDIR /app

ADD tools tools

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd tools/ && go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

ADD . .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -gcflags="all=-N -l" -mod=readonly -o /guardiand github.com/certusone/wormhole/bridge

ENTRYPOINT /guardiand
