# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:lts-alpine@sha256:2ae9624a39ce437e7f58931a5747fdc60224c6e40f8980db90728de58e22af7c

RUN mkdir -p /app
WORKDIR /app

ADD . .

# create .env files from .env.sample, if they do not already exist.
RUN [[ ! -f .env.development ]] && cp .env.sample .env.development
RUN [[ ! -f .env.production ]] && cp .env.sample .env.production

RUN npm ci
