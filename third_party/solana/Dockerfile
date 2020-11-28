# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44

# Depend on our smart contract build - it's going to be linked directly into Solana as part the patch we carry.
FROM solana-contract as contract

FROM rust:1.45

RUN apt-get update && apt-get install -y libssl-dev libudev-dev pkg-config zlib1g-dev llvm clang
RUN rustup component add rustfmt

WORKDIR /usr/src/solana

RUN git clone https://github.com/solana-labs/solana --branch master && \
  cd solana && git checkout v1.4.7

ADD *.patch .

RUN cd solana && patch -p0 < ../Add_crypto_syscalls.patch

RUN --mount=type=cache,target=/usr/local/cargo,from=rust,source=/usr/local/cargo \
    --mount=type=cache,target=solana/target \
    cd solana && \
    cargo build --release && \
    mkdir -p /opt/solana/deps && \
    cp $(find ./target/release -maxdepth 1 -type f ! -name "*.*") /opt/solana && \
    cp ./target/release/deps/*.so /opt/solana/deps

COPY --from=contract /opt/solana/deps/spl_bridge.so /opt/solana/deps/spl_bridge.so

ENV PATH="/opt/solana:${PATH}"
ENV RUST_LOG="solana_runtime::system_instruction_processor=trace,solana_runtime::message_processor=trace,solana_bpf_loader=debug,solana_rbpf=debug"

CMD ["/usr/src/solana/solana/run.sh"]
