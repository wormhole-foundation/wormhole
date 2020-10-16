# syntax=docker/dockerfile:experimental
FROM golang:1.15.3

WORKDIR /app

ADD . .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -mod=readonly -o /guardiand github.com/certusone/wormhole/bridge/cmd/guardiand

ENTRYPOINT /guardiand
