FROM docker.io/golang:1.19.9@sha256:86901bd215ca2b33bdef68f70b999140252169bb55883bb29347334fa00b5b62 AS bitcoin-build

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

