# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM node:19.6.1-slim@sha256:a1ba21bf0c92931d02a8416f0a54daad66cb36a85d2b73af9d73b044f5f57cfc

# npm wants to clone random Git repositories - lovely.
# RUN apk add git python make build-base
# RUN apk update && apk add bash
RUN apt-get update && apt-get -y install \
  git python make curl netcat

RUN npm i typescript -g
RUN curl -L https://foundry.paradigm.xyz | bash 
RUN $HOME/.foundry/bin/foundryup
RUN ls $HOME/.foundry/bin

# Run as user, otherwise, npx explodes.
RUN mv /root/.foundry/bin/forge /bin/forge
USER 1000


RUN mkdir -p /home/node/app
RUN mkdir -p /home/node/.npm

WORKDIR /home/node/app

# Fix git ssh error
RUN git config --global url."https://".insteadOf ssh://

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Node
ENV NODE_EXTRA_CA_CERTS=/certs/cert.pem
ENV NODE_OPTIONS=--use-openssl-ca
# npm
RUN if [ -e /certs/cert.pem ]; then npm config set cafile /certs/cert.pem; fi
# git
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi


WORKDIR /home/node/app

# Only invalidate the npm install step if package.json changed
COPY --chown=node:node package.json .
COPY --chown=node:node package-lock.json .
COPY --chown=node:node .env.test .env


# We want to cache node_modules *and* incorporate it into the final image.
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  --mount=type=cache,uid=1000,gid=1000,target=node_modules \
  npm ci && \                                                 
  cp -R node_modules node_modules_cache
  

# Amusingly, Debian's coreutils version has a bug where mv believes that
# the target is on a different fs and does a full recursive copy for what
# could be a renameat syscall. Alpine does not have this bug.
RUN rm -rf node_modules && mv node_modules_cache node_modules

COPY --chown=node:node . .
RUN make build

ARG num_guardians
ENV NUM_GUARDIANS=$num_guardians
