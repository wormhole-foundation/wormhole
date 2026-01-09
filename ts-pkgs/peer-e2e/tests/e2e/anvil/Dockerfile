FROM ghcr.io/foundry-rs/foundry:v1.5.1@sha256:7ec8952cc5322dce65091768e9efab8641ea9b54105f21fd71d4ae3dc3da05a8 as foundry
FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get update && apt-get -y install git make

COPY --from=foundry /usr/local/bin/anvil /usr/local/bin/forge /usr/local/bin/cast /bin/

# TODO: Pin the commit
RUN git clone -b feat/dkg-docker --depth 1 https://github.com/XLabs/core-bridge.git

WORKDIR /core-bridge

RUN make build-evm

WORKDIR /core-bridge/ts-pkgs/peer-e2e/tests/e2e/anvil/

ENTRYPOINT ["./localAnvilWithVerifier.sh"]
