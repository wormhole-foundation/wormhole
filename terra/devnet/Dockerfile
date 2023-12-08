FROM --platform=linux/amd64 docker.io/golang:1.20.10-bullseye@sha256:082569b3303b164cc4a7c88ac59b19b69c1a5d662041ac0dca046ac239632442 AS build

WORKDIR /root/core

RUN git init
RUN git remote add origin https://github.com/classic-terra/core.git
RUN git fetch --depth 1 origin 2a8b6db5a0235c29dcc714c48d36387fc0d1d793
RUN git checkout FETCH_HEAD

RUN make install

COPY config /root/.terra/config
COPY data /root/.terra/data

EXPOSE 26657
EXPOSE 1317
EXPOSE 9090
EXPOSE 9091
