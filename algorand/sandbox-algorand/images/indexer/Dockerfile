ARG GO_VERSION=1.18.2
FROM golang:$GO_VERSION-alpine

# Support additional root CAs
COPY config.dev cert.pem* /certs/
# Debian (for top Go priority)
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
# Alpine
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/cert.pem; fi

# Environment variables used by install.sh
ARG URL=https://github.com/algorand/indexer
ARG BRANCH=master
ARG SHA=""

ENV HOME /opt/indexer
WORKDIR /opt/indexer

ENV DEBIAN_FRONTEND noninteractive
RUN apk add --no-cache git bzip2 make bash libtool boost-dev autoconf automake g++ postgresql

# Support additional root CAs
# git
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

# Copy files to container.
COPY images/indexer/disabled.go /tmp/disabled.go
COPY images/indexer/start.sh /tmp/start.sh
COPY images/indexer/install.sh /tmp/install.sh

# Install indexer binaries.
RUN /tmp/install.sh

CMD ["/tmp/start.sh"]
