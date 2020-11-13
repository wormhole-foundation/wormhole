# syntax=docker/dockerfile:experimental

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:lts-alpine

USER 1000
RUN mkdir -p /home/node/app
RUN mkdir -p /home/node/.npm
WORKDIR /home/node/app

ADD --chown=node:node package.json .
ADD --chown=node:node package-lock.json .

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  --mount=type=cache,uid=1000,gid=1000,target=node_modules \
  npm install && \
  cp -r node_modules node_modules_cache

RUN rmdir node_modules && mv node_modules_cache node_modules

ADD --chown=node:node . .
