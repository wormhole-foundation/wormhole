# There's nothing special about this version, it is simply the `latest` as of
# the creation date of this file.
FROM alpine:3.20.3@sha256:1e42bbe2508154c9126d48c2b8a75420c3544343bf86fd041fb7527e017a4b4a

RUN apk add --no-cache inotify-tools

COPY monitor.sh /monitor.sh
RUN chmod +x /monitor.sh
