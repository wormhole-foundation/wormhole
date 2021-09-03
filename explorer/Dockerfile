# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:16-alpine@sha256:004dbac84fed48e20f9888a23e32fa7cf83c2995e174a78d41d9a9dd1e051a20

RUN mkdir -p /app
WORKDIR /app

ADD package.json .
ADD package-lock.json .
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm ci

ADD . .

# create .env files from .env.sample, if they do not already exist.
RUN [[ ! -f .env.development ]] && cp .env.sample .env.development
RUN [[ ! -f .env.production ]] && cp .env.sample .env.production

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm run build
