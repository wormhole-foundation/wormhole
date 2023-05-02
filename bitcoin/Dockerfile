FROM docker.io/golang:1.19.8@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8b4e2aeb08f7ab739292b3 AS bitcoin-build

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

