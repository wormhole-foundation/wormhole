FROM docker.io/golang:1.19@sha256:7ffa70183b7596e6bc1b78c132dbba9a6e05a26cd30eaa9832fecad64b83f029 AS bitcoin-build

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

