FROM docker.io/library/rust:1.70.0-slim-buster@sha256:9e6d3a85a45821bb670b745960a0b3fb4bbbf550c45af6000423348e0e4685e3

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat jq curl

RUN curl -OL --ssl https://github.com/informalsystems/hermes/archive/refs/tags/v1.6.0.tar.gz
RUN printf '%s %s\n' "1675ccd585635d4f934dc2cd1153b20db85f8dcbdf382be5ae65983f5bcf6d6e" "v1.6.0.tar.gz" | sha256sum --check
RUN tar xzf v1.6.0.tar.gz
WORKDIR hermes-1.6.0
RUN cargo build --release --bin hermes --locked
RUN mv target/release/hermes /usr/bin

WORKDIR /app

COPY ./wormchain/ibc-relayer/mnemonic.txt ./
COPY ./wormchain/ibc-relayer/hermes-config.toml /root/.hermes/config.toml
COPY ./wormchain/ibc-relayer/hermes-startup.sh ./

RUN chmod +x ./hermes-startup.sh

EXPOSE 3000
EXPOSE 3001