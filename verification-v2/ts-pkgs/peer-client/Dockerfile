FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN mkdir --parents core-bridge/ts-pkgs/peer-client core-bridge/ts-pkgs/peer-lib
COPY --link .yarn /core-bridge/.yarn
COPY --link package.json yarn.lock .yarnrc.yml /core-bridge/
COPY --link ts-pkgs/peer-client/package.json /core-bridge/ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-lib/package.json /core-bridge/ts-pkgs/peer-lib/
WORKDIR /core-bridge
RUN yarnpkg workspaces focus --all

COPY --link ts-pkgs/config/ ts-pkgs/config/
COPY --link ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY --link ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY --link ts-pkgs/peer-client/tsconfig.json ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-client/src ts-pkgs/peer-client/src
WORKDIR /core-bridge/ts-pkgs/peer-client
RUN yarnpkg build

ARG TLS_HOSTNAME
ARG TLS_PORT
ARG PEER_SERVER_URL
# Provide exactly one of GUARDIAN_PRIVATE_KEY_ARN or mount guardian_pk secret
ARG GUARDIAN_PRIVATE_KEY_ARN

RUN test -n "${TLS_HOSTNAME}" && \
    test -n "${TLS_PORT}" && \
    test -n "${PEER_SERVER_URL}"

# Generate config: use ARN if provided, otherwise use the secret file path
RUN if [ -n "${GUARDIAN_PRIVATE_KEY_ARN}" ]; then \
      echo '{"guardianPrivateKeyArn":"'"${GUARDIAN_PRIVATE_KEY_ARN}"'","serverUrl":"'"${PEER_SERVER_URL}"'","peer":{"hostname":"'"${TLS_HOSTNAME}"'","port":'"${TLS_PORT}"',"tlsX509":"/run/secrets/cert.pem"}}' > self_config.json; \
    else \
      echo '{"guardianPrivateKeyPath":"/run/secrets/guardian_pk","serverUrl":"'"${PEER_SERVER_URL}"'","peer":{"hostname":"'"${TLS_HOSTNAME}"'","port":'"${TLS_PORT}"',"tlsX509":"/run/secrets/cert.pem"}}' > self_config.json; \
    fi

# The cert is not really a secret, but it doesn't hurt to use the same mount type for both
# When using ARN, guardian_pk secret is not required; AWS credentials are passed for KMS
RUN --mount=type=secret,id=guardian_pk,required=false \
    --mount=type=secret,id=cert.pem \
    --mount=type=secret,id=aws_access_key_id,required=false \
    --mount=type=secret,id=aws_secret_access_key,required=false \
    --mount=type=secret,id=aws_session_token,required=false \
    AWS_ACCESS_KEY_ID=$(cat /run/secrets/aws_access_key_id 2>/dev/null || true) \
    AWS_SECRET_ACCESS_KEY=$(cat /run/secrets/aws_secret_access_key 2>/dev/null || true) \
    AWS_SESSION_TOKEN=$(cat /run/secrets/aws_session_token 2>/dev/null || true) \
    yarnpkg start:client upload