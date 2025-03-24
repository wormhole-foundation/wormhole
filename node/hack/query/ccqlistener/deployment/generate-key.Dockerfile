FROM golang:1.20-bullseye

WORKDIR /app

COPY key-generator.go .

RUN go mod init keygen && \
    go get github.com/ethereum/go-ethereum@v1.10.26 && \
    go build -o keygen

ENTRYPOINT ["/app/keygen"] 