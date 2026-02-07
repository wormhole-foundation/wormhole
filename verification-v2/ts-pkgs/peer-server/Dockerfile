FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN mkdir --parents verification-v2/ts-pkgs/peer-server verification-v2/ts-pkgs/peer-lib
COPY .yarn /verification-v2/.yarn
COPY package.json yarn.lock .yarnrc.yml /verification-v2/
COPY ts-pkgs/peer-server/package.json /verification-v2/ts-pkgs/peer-server/
COPY ts-pkgs/peer-lib/package.json /verification-v2/ts-pkgs/peer-lib/
WORKDIR /verification-v2
RUN yarnpkg workspaces focus --all

COPY ts-pkgs/config/ ts-pkgs/config/
COPY ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY ts-pkgs/peer-server/tsconfig.json ts-pkgs/peer-server/
COPY ts-pkgs/peer-server/src ts-pkgs/peer-server/src

WORKDIR /verification-v2/ts-pkgs/peer-server
RUN yarnpkg build

# The config for the peer server is expected at /verification-v2/ts-pkgs/peer-server/config.json

ENTRYPOINT [ \
  "node", \
  "--enable-source-maps", \
  "--require", "/verification-v2/.pnp.cjs", \
  "--loader", "/verification-v2/.pnp.loader.mjs", \
  "/verification-v2/ts-pkgs/peer-server/ts-build/src/cli.js" \
]