FROM rust:1.62@sha256:5777f201f507075309c4d2d1c1e8d8219e654ae1de154c844341050016a64a0c as aptos-node

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Debian
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
# git
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

RUN apt-get update && apt-get -y install libclang-dev jq cmake curl python3 --no-install-recommends

COPY node_builder.sh /tmp

WORKDIR /tmp

RUN ./node_builder.sh

COPY start_node.sh  /tmp
