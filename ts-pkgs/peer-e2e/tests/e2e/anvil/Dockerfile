FROM ghcr.io/foundry-rs/foundry:v1.5.1@sha256:7ec8952cc5322dce65091768e9efab8641ea9b54105f21fd71d4ae3dc3da05a8 AS foundry
FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get update && apt-get -y install git make

COPY --from=foundry /usr/local/bin/anvil /usr/local/bin/forge /usr/local/bin/cast /bin/

RUN mkdir --parents core-bridge/src
COPY --link foundry.toml Makefile core-bridge/
COPY --link src/evm core-bridge/src/evm
COPY --link test core-bridge/test
COPY --link ts-pkgs/peer-e2e/tests/e2e/anvil/localAnvilWithVerifier.sh core-bridge/

WORKDIR /core-bridge

RUN make build-evm

ENTRYPOINT ["./localAnvilWithVerifier.sh"]
