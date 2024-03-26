ARG GO_IMAGE_VERSION=1.21.1-bullseye
ARG GO_IMAGE_DIGEST=18f57bdd1d188ade1226471393b5c3ced4639c6c772fd94853ea7abd9eb158f7
FROM docker.io/golang:${GO_IMAGE_VERSION}@sha256:${GO_IMAGE_DIGEST} AS algorand-algod

ARG CHANNEL=master
ARG URL="https://github.com/algorand/go-algorand"
ARG BRANCH=master
ARG SHA=

# When these are set attempt to connect to a network.
ARG GENESIS_FILE=""
ARG BOOTSTRAP_URL=""

# Options for algod config
ARG ALGOD_PORT="4001"
ARG KMD_PORT="4002"
ARG TOKEN="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
ARG TEMPLATE="images/algod/DevModeNetwork.json"

RUN echo "Installing from source. ${URL} -- ${BRANCH}"
ENV BIN_DIR="$HOME/node"
ENV ALGORAND_DATA="/opt/data"

# Basic dependencies.
ENV HOME /opt
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install -y apt-utils curl git git-core bsdmainutils python3

# Copy lots of things into the container. The gitignore indicates which directories.
COPY . /tmp

# Install algod binaries.
RUN /tmp/images/algod/install.sh \
    -d "${BIN_DIR}" \
    -u "https://github.com/algorand/go-algorand" \
    -b "v3.16.2-stable" \
    -s ""

# Configure network
RUN /tmp/images/algod/setup.py \
 --bin-dir "$BIN_DIR" \
 --data-dir "/opt/data" \
 --start-script "/opt/start_algod.sh" \
 --network-dir "/opt/testnetwork" \
 --network-template "//tmp/images/algod/DevModeNetwork.json" \
 --network-token "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
 --algod-port "4001" \
 --kmd-port "4002" \
 --bootstrap-url "${BOOTSTRAP_URL}" \
 --genesis-file "/tmp/${GENESIS_FILE}"


ENV PATH="$BIN_DIR:${PATH}"
WORKDIR /opt/data

# Start algod
CMD ["/opt/start_algod.sh"]
