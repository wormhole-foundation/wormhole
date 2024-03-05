# Use a specific version of the Go base image
FROM docker.io/golang:1.19.9@sha256:86901bd215ca2b33bdef68f70b999140252169bb55883bb29347334fa00b5b62 AS bitcoin-build

# Set the architecture argument
ARG ARCH=amd64

# Enable Go modules
ENV GO111MODULE=on

# Set the working directory
WORKDIR /app

# Update packages and install git and netcat in a single layer
RUN apt-get update && apt-get install -y git netcat && rm -rf /var/lib/apt/lists/*

# Clone btcd and install
RUN git clone https://github.com/btcsuite/btcd.git && \
    WORKDIR /app/btcd && \
    go install -v . ./cmd/... && \
    WORKDIR /app

# Clone btcwallet and install
RUN git clone https://github.com/btcsuite/btcwallet.git && \
    WORKDIR /app/btcwallet && \
    go install -v . ./cmd/... && \
    WORKDIR /app

# Copy scripts and change to root directory
WORKDIR /root
COPY scripts/.btcwallet .btcwallet
COPY scripts/start_node.sh .
