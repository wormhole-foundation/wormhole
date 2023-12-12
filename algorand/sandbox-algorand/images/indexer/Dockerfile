ARG GO_VERSION=1.18.2
FROM golang:$GO_VERSION-alpine@sha256:4795c5d21f01e0777707ada02408debe77fe31848be97cf9fa8a1462da78d949

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
