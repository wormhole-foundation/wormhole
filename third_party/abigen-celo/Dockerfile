FROM golang:1.17.9-bullseye@sha256:5e415dc60e1277bd0fa3bd9f978ca58c8cf82ec6b6e0a7d67c2d1900e77039e2

RUN apt update && apt install curl git gcc libc-dev

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go build -o /bin/abigen github.com/celo-org/celo-blockchain/cmd/abigen
