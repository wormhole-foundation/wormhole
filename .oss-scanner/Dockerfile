# syntax=docker/dockerfile:1
#
# Build environment for the Anthropic OSS Scanner (https://github.com/anthropics/oss-scanner).
#
# This is NOT a runtime image and it does not start a guardian, a devnet or any chain. The scanner builds
# this file with the repository as the build context, adds its own tooling layer on top, and then runs
# an agent inside the image with no network access. Everything that the agent needs to build and test the
# in-scope code must therefore be fetched here, while the network is still available.
#
# In scope (see .oss-scanner/threat_model.md): the guardian node (node/) and the Go SDK it depends on
# (sdk/). The on-chain contracts in this repository are not built or scanned by this image.
#
# wormchain/ is also copied in and its modules fetched, because node/go.mod replaces the wormchain module
# with that directory; it is a build dependency of the guardian, not itself a target.
#
# Local check (from a clone of anthropics/oss-scanner): tools/check wormhole
# Or directly:  docker build -f .oss-scanner/Dockerfile .

# Keep in sync with node/Dockerfile and node/go.mod.
FROM docker.io/library/golang:1.26.10-bookworm@sha256:fe3da92a5cb2881e4aa75e625cb0b7c50571632fb4cf340cbe53e6ccdcffdd51

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates build-essential git make curl jq python3 \
    && rm -rf /var/lib/apt/lists/*

# Never try to download a Go toolchain: the image's Go matches node/go.mod.
ENV GOTOOLCHAIN=local \
    GOFLAGS=-mod=readonly

# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src

# Fetch every module the guardian, the SDK and wormchain need, so nothing is downloaded during the scan.
RUN cd node && go mod download \
    && cd ../sdk && go mod download \
    && cd ../wormchain && go mod download

# Build guardiand the way the Makefile does (build/bin/guardiand), then compile every package and every
# test binary so the Go build cache is warm: `go test ./...` then only has to run, not rebuild, on the
# scanner's small agent machine. The -race variant is compiled too, since make test-coverage uses it.
RUN make node \
    && cd node && go build ./... && go test -run '^$' ./... && go test -race -run '^$' ./... \
    && cd ../sdk && go build ./... && go test -run '^$' ./... && go test -race -run '^$' ./...

# wasmvm is a C library shipped inside its Go module; install it system-wide like node/Dockerfile does so
# any binary linked against it also runs outside the module cache.
RUN cp "$(go env GOMODCACHE)"/github.com/!cosm!wasm/wasmvm@v1.1.1/internal/api/libwasmvm.x86_64.so /usr/lib/ 2>/dev/null \
    || cp "$(go env GOMODCACHE)"/github.com/!cosm!wasm/wasmvm@v1.1.1/internal/api/libwasmvm.aarch64.so /usr/lib/

# Cheap smoke test so a broken image is noticed at build time. A failing test must not abort the build.
RUN (cd sdk && go test -short ./...) || echo "WARNING: sdk tests reported failures"
RUN (cd node && go test -short ./pkg/processor/... ./pkg/governor/... ./pkg/p2p/...) \
    || echo "WARNING: node tests reported failures"
