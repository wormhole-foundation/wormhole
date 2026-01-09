FROM ghcr.io/foundry-rs/foundry:v1.3.4@sha256:3afb57dcd8f06e098d643d04e0b541ef83a1edf94c0a80ea5e89329ec50ccd92 AS builder

# Foundry image runs as foundry user by default
# We need root to both run apt and to write files to the filesystem
USER root
RUN apt update && apt install -y jq wget

# Preflight

WORKDIR /app
COPY foundry.toml foundry.toml
COPY --link lib/wormhole-solidity-sdk lib/wormhole-solidity-sdk
COPY --link src/evm src/evm

# Prepare compiler input. NOTE: jq must be pre-applied to "clean" the output from forge.
# Otherwise solc aborts with duplicated key/newline problems.

RUN forge verify-contract --show-standard-json-input 0x0000000000000000000000000000000000000000 src/evm/WormholeVerifier.sol:WormholeVerifier | jq '.' > WormholeVerifier.input.json

# Get compiler according to forge configuration (foundry.toml specified)

RUN SOLC_VERSION=$(forge config | grep "^solc =" | sed 's/solc = //' | sed 's/"//g'); \
    if [ -z $SOLC_VERSION ]; then echo "SOLC_VERSION not set"; exit 1; fi; \
    wget --output-document=solc https://github.com/ethereum/solidity/releases/download/v$SOLC_VERSION/solc-static-linux && chmod +x solc

# Compile contract(s).

RUN ./solc --standard-json WormholeVerifier.input.json > WormholeVerifier.output.json && \
    SOLC_ERR=$(jq '.errors[]? | select(.severity == "error")' WormholeVerifier.output.json) && \
    if [ ! -z "$SOLC_ERR" ]; then \
        echo "Error detected during solc execution."; \
        echo $SOLC_ERR; \
        exit 2; \
    fi

# Consolidate all generated output
FROM scratch AS foundry-export
COPY --from=builder /app/*.input.json /app/*.output.json ./
