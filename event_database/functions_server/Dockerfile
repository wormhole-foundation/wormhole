# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44
FROM docker.io/golang:1.17.0@sha256:06e92e576fc7a7067a268d47727f3083c0a564331bfcbfdde633157fc91fb17d

WORKDIR /app

COPY . .

WORKDIR /app/functions_server
RUN --mount=type=cache,target=/root/.cache --mount=type=cache,target=/go \
    go build -mod=readonly -o /functions main.go

CMD ["/functions"]
