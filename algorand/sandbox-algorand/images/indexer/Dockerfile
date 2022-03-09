ARG GO_VERSION=1.17.5
FROM golang:$GO_VERSION-alpine

# Environment variables used by install.sh
ARG URL=https://github.com/algorand/indexer
ARG BRANCH=master
ARG SHA=""

ENV HOME /opt/indexer
WORKDIR /opt/indexer

ENV DEBIAN_FRONTEND noninteractive
RUN apk add --no-cache git bzip2 make bash libtool boost-dev autoconf automake g++

# Copy files to container.
COPY images/indexer/disabled.go /tmp/disabled.go
COPY images/indexer/start.sh /tmp/start.sh
COPY images/indexer/install.sh /tmp/install.sh

# Install indexer binaries.
RUN /tmp/install.sh

CMD ["/tmp/start.sh"]
