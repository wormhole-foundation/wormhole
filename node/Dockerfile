# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM --platform=linux/amd64 docker.io/golang:1.19.3@sha256:dc76ef03e54c34a00dcdca81e55c242d24b34d231637776c4bb5c1a8e8514253 AS build
# libwasmvm.so is not compatible with arm

# Support additional root CAs
COPY node/go.mod cert.pem* /certs/
# Debian
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
# git
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

WORKDIR /app

COPY node/tools tools

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd tools/ && go build -mod=readonly -o /dlv github.com/go-delve/delve/cmd/dlv

COPY node node
COPY sdk sdk

ARG GO_BUILD_ARGS=-race

RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
  cd node && \
  go build ${GO_BUILD_ARGS} -gcflags="all=-N -l" --ldflags '-extldflags "-Wl,--allow-multiple-definition" -X "github.com/certusone/wormhole/node/cmd/guardiand.Build=dev"' -mod=readonly -o /guardiand github.com/certusone/wormhole/node

# Only export the final binary (+ shared objects). This reduces the image size
# from ~1GB to ~150MB.
FROM scratch as export

# guardiand can't (easily) be statically linked due to the C dependencies, so we
# have to copy all the dynamic libraries
COPY --from=build /lib/* /lib/
COPY --from=build /lib64/* /lib64/

# Copy the shells as entrypoints, but no utilities are necessary
COPY --from=build /bin/bash /bin/dash /bin/sh /bin/

# finally copy the guardian executable
COPY --from=build /guardiand .

ENTRYPOINT ["/guardiand"]
