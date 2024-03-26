ARG GO_IMAGE_VERSION=1.21.1-alpine
ARG GO_IMAGE_DIGEST=1c9cc949513477766da12bfa80541c4f24957323b0ee00630a6ff4ccf334b75b
FROM docker.io/golang:${GO_IMAGE_VERSION}@sha256:${GO_IMAGE_DIGEST}

# Environment variables used by install.sh
ARG URL=https://github.com/algorand/indexer
ARG BRANCH=2.15.4
ARG SHA=""

ENV HOME /opt/indexer
WORKDIR /opt/indexer

ENV DEBIAN_FRONTEND noninteractive
RUN apk add --no-cache git bzip2 make bash libtool boost-dev autoconf automake g++ postgresql

# Copy files to container.
COPY images/indexer/disabled.go /tmp/disabled.go
COPY images/indexer/start.sh /tmp/start.sh
COPY images/indexer/install.sh /tmp/install.sh

# Install indexer binaries.

RUN /tmp/install.sh

CMD ["/tmp/start.sh"]
