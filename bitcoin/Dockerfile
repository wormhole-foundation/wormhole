FROM docker.io/golang:1.19.3@sha256:dc76ef03e54c34a00dcdca81e55c242d24b34d231637776c4bb5c1a8e8514253 AS bitcoin-build

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

