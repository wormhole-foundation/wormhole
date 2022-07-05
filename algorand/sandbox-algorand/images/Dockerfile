FROM ubuntu:20.04@sha256:fd92c36d3cb9b1d027c4d2a72c6bf0125da82425fc2ca37c414d4f010180dc19
ARG channel

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get install -y ca-certificates curl

# Use a non-privilidged user with a random UID / GID for security reasons
RUN groupadd -g 10353 algorand && \
  useradd -m -u 10353 -g algorand algorand && \
  chown -R algorand:algorand /opt && \
  ls -lha /opt

USER algorand

COPY --chown=algorand:algorand ./config/update.sh /tmp

RUN \
  set -eux; \
  mkdir /opt/installer ; \
  cd /opt/installer ; \
  mv /tmp/update.sh . ; \
  ./update.sh -i -c $channel -p /opt/algorand/node -d /opt/algorand/node/data.tmp -n ; \
  rm -rf /opt/algorand/node/data.tmp ; \
  mkdir /opt/algorand/node/data

COPY ./config/start.sh /opt/algorand

VOLUME /opt/algorand/node/data

# Set up environment variable to make life easier
ENV PATH="/opt/algorand/node:${PATH}"
ENV ALGORAND_DATA="/opt/algorand/node/data"

ENTRYPOINT [ "/opt/algorand/start.sh" ]
