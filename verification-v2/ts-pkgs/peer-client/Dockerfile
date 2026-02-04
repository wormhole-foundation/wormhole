FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN mkdir --parents verification-v2/ts-pkgs/peer-client verification-v2/ts-pkgs/peer-lib
COPY --link .yarn /verification-v2/.yarn
COPY --link package.json yarn.lock .yarnrc.yml /verification-v2/
COPY --link ts-pkgs/peer-client/package.json /verification-v2/ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-lib/package.json /verification-v2/ts-pkgs/peer-lib/
WORKDIR /verification-v2
RUN yarnpkg workspaces focus --all

COPY --link ts-pkgs/config/ ts-pkgs/config/
COPY --link ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY --link ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY --link ts-pkgs/peer-client/tsconfig.json ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-client/src ts-pkgs/peer-client/src
WORKDIR /verification-v2/ts-pkgs/peer-client
RUN yarnpkg build

ARG TLS_HOSTNAME
ARG TLS_PORT
ARG PEER_SERVER_URL
# Provide exactly one of GUARDIAN_KMS_ARN or mount guardian_pk secret
ARG GUARDIAN_KMS_ARN

RUN test -n "${TLS_HOSTNAME}" && \
    test -n "${TLS_PORT}" && \
    test -n "${PEER_SERVER_URL}"

# Generate config: use ARN if provided, otherwise use the secret file path
RUN if [ -n "${GUARDIAN_KMS_ARN}" ]; then \
      echo '{"guardianPrivateKeyArn":"'"${GUARDIAN_KMS_ARN}"'","serverUrl":"'"${PEER_SERVER_URL}"'","peer":{"hostname":"'"${TLS_HOSTNAME}"'","port":'"${TLS_PORT}"',"tlsX509":"/run/secrets/cert.pem"}}' > self_config.json; \
    else \
      echo '{"guardianPrivateKeyPath":"/run/secrets/guardian_pk","serverUrl":"'"${PEER_SERVER_URL}"'","peer":{"hostname":"'"${TLS_HOSTNAME}"'","port":'"${TLS_PORT}"',"tlsX509":"/run/secrets/cert.pem"}}' > self_config.json; \
    fi

# When using AWS KMS, the guardian_pk secret is not required.
# AWS credentials should be assigned to the IAM role running the container.
# The client will then query ephemeral credentials and sign with those.
# When using a raw private key file, the mount should look like this:
# --volume "<src file>:/run/secrets/guardian_pk,ro"
# The cert is not really a secret, but it doesn't hurt to use this mount type.
# --volume "<src file>:/run/secrets/cert.pem,ro"
ENTRYPOINT ["yarnpkg", "start:client", "upload"]