# syntax=docker/dockerfile:experimental
FROM golang:1.14

WORKDIR /app

ADD . .

RUN --mount=type=cache,target=/root/.cache/go-build --mount=type=cache,target=/go/pkg \
  go build -o /guardiand github.com/certusone/wormhole/bridge/cmd/guardiand

ENTRYPOINT /guardiand
