FROM --platform=linux/amd64 docker.io/golang:1.17.5@sha256:90d1ab81f3d157ca649a9ff8d251691b810d95ea6023a03cdca139df58bca599 AS bitcoin-build

ARG ARCH=amd64

ENV GO111MODULE=on

WORKDIR /app

RUN apt-get update && apt-get install git netcat

RUN git clone https://github.com/btcsuite/btcd.git

WORKDIR /app/btcd

RUN go install -v . ./cmd/...

WORKDIR /app

RUN git clone https://github.com/btcsuite/btcwallet.git
WORKDIR /app/btcwallet

RUN go install -v . ./cmd/...

WORKDIR /root
COPY scripts/.btcwallet .btcwallet
COPY scripts/start_node.sh .

