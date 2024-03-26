ARG GO_IMAGE_VERSION=1.21.1-bullseye
ARG GO_IMAGE_DIGEST=18f57bdd1d188ade1226471393b5c3ced4639c6c772fd94853ea7abd9eb158f7
FROM --platform=linux/amd64 docker.io/golang:${GO_IMAGE_VERSION}@sha256:${GO_IMAGE_DIGEST} AS build

WORKDIR /root/core

RUN git init
RUN git remote add origin https://github.com/classic-terra/core.git
RUN git fetch --depth 1 origin 2a8b6db5a0235c29dcc714c48d36387fc0d1d793
RUN git checkout FETCH_HEAD

RUN make install

COPY config /root/.terra/config
COPY data /root/.terra/data

EXPOSE 26657
EXPOSE 1317
EXPOSE 9090
EXPOSE 9091
