# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM docker.io/golang:1.18.1@sha256:3f0168c019343d7cc07bf2481e7b6555fbf2ebadfb01f9e77875da66439ba041

# Support additional root CAs
COPY go.mod cert.pem* /certs/
# Debian
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

WORKDIR /app

ADD tools tools

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd tools/ && go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

ADD . .

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  go build -race -gcflags="all=-N -l" -mod=readonly -o /guardiand github.com/certusone/wormhole/node

ENTRYPOINT /guardiand
