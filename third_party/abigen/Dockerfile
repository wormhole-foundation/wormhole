FROM golang:1.15.6-alpine@sha256:49b4eac11640066bc72c74b70202478b7d431c7d8918e0973d6e4aeb8b3129d2

RUN apk add curl git gcc libc-dev linux-headers

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go build -mod=readonly -o /bin/abigen github.com/ethereum/go-ethereum/cmd/abigen
