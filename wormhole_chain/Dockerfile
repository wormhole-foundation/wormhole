# syntax = docker/dockerfile:1.2
# WARNING! Use `DOCKER_BUILDKIT=1` with `docker build` to enable --mount feature.

## prep the base image.
#
FROM docker.io/golang:1.17.5@sha256:90d1ab81f3d157ca649a9ff8d251691b810d95ea6023a03cdca139df58bca599

# RUN apt update && \
#     apt-get install -y \
#         build-essential \
#         ca-certificates \
#         curl

# # enable faster module downloading.
# ENV GOPROXY https://proxy.golang.org

WORKDIR /app

COPY . .

RUN curl https://get.starport.network/starport! | bash

EXPOSE 26657
EXPOSE 26656
EXPOSE 6060 
EXPOSE 9090 
EXPOSE 1317 

RUN unset GOPATH

RUN starport chain build --output build
WORKDIR ./build

#TODO allow moniker from args
# RUN ./wormhole-chaind init bootstrapValidator --chain-id wormhole-chain-devnet
RUN ["/bin/bash", "-c", "./wormhole-chaind init bootstrapValidator --chain-id wormhole-chain-devnet"]

# #TODO allow config file changes here

ENTRYPOINT ["/bin/bash","-c","./wormhole-chaind start"]
