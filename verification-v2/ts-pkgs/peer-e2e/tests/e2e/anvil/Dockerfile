FROM ghcr.io/foundry-rs/foundry:v1.5.1@sha256:3a70bfa9bd2c732a767bb60d12c8770b40e8f9b6cca28efc4b12b1be81c7f28e AS foundry
FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get --quiet update && apt-get --quiet --no-install-recommends --yes install git make ca-certificates

COPY --from=foundry /usr/local/bin/anvil /usr/local/bin/forge /usr/local/bin/cast /bin/

RUN mkdir --parents /verification-v2/src
COPY foundry.toml Makefile /verification-v2/
COPY src/evm /verification-v2/src/evm
COPY test /verification-v2/test
COPY ts-pkgs/peer-e2e/tests/e2e/anvil/localAnvilWithVerifier.sh /verification-v2/

WORKDIR /verification-v2

RUN make build-evm

ENTRYPOINT ["./localAnvilWithVerifier.sh"]
