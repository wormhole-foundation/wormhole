FROM ghcr.io/wormhole-foundation/aptos:0.3.3@sha256:56fbcbf5caf7a4d24dee373b57a4bf80880b7500bdad642454f07a766c4084c8 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

FROM aptos as tests

WORKDIR /tmp

COPY contracts/ contracts
COPY deployer/ deployer
COPY Makefile Makefile

RUN make test
