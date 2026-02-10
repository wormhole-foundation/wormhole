FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get --quiet update && apt-get --quiet --no-install-recommends --yes install \
  git \
  golang \
  jq \
  ca-certificates \
  && rm -rf /var/lib/apt/lists

# TODO: Pin the commit
RUN git clone --revision=0d03be8ea6bb78d8133eb6e8f48a718ddbdcea62 --depth 1 https://github.com/XLabs/tss-lib.git

WORKDIR /tss-lib/tss/internal/cmd
RUN go build -o=./server ./dkg

WORKDIR /
RUN mkdir --parents verification-v2/ts-pkgs/peer-client verification-v2/ts-pkgs/peer-lib
COPY .yarn verification-v2/.yarn
COPY package.json yarn.lock .yarnrc.yml verification-v2/
COPY ts-pkgs/peer-client/package.json verification-v2/ts-pkgs/peer-client/
COPY ts-pkgs/peer-lib/package.json verification-v2/ts-pkgs/peer-lib/
WORKDIR /verification-v2
RUN yarnpkg workspaces focus --all

COPY ts-pkgs/config/ ts-pkgs/config/
COPY ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY ts-pkgs/peer-client/tsconfig.json ts-pkgs/peer-client/
COPY ts-pkgs/peer-client/src ts-pkgs/peer-client/src
WORKDIR /verification-v2/ts-pkgs/peer-client
RUN yarnpkg build

COPY --chmod=555 <<EOT poll_guardians.sh
#!/bin/bash

set -euo pipefail

node --enable-source-maps \\
  --require /verification-v2/.pnp.cjs \\
  --loader /verification-v2/.pnp.loader.mjs \\
  /verification-v2/ts-pkgs/peer-client/ts-build/src/cli.js poll

jq '.StorageLocation = "/keys/"' peer_config.json > /keys/dkg_config.json

exec /tss-lib/tss/internal/cmd/server -cnfg=/keys/dkg_config.json -protocol=FROST:DKG -sk=/keys/key.pem

EOT

ENTRYPOINT ["./poll_guardians.sh"]
