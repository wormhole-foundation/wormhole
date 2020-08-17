FROM golang:1.14.7-alpine

RUN apk add curl git gcc libc-dev linux-headers

WORKDIR /app

ADD go.mod .
ADD go.sum .

RUN go build -mod=readonly -o /bin/abigen github.com/ethereum/go-ethereum/cmd/abigen
