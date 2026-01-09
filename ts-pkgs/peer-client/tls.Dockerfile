FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get --quiet update && apt-get --quiet --no-install-recommends --yes install \
  openssl \
  && rm -rf /var/lib/apt/lists

# Generate the TLS key and certificate
COPY --chmod=555 <<EOT /generate_tls_key.sh
#!/bin/bash
set -euo pipefail

if [ -z "\${TLS_HOSTNAME}" ]; then
  echo "TLS_HOSTNAME is not set"
  exit 1
fi
if [ -z "\${TLS_PUBLIC_IP}" ]; then
  echo "TLS_PUBLIC_IP is not set"
  exit 1
fi

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out /keys/key.pem
openssl req -x509 -key /keys/key.pem -out /keys/cert.pem -days 365 \\
  -subj "/CN=\${TLS_HOSTNAME}" \\
  -addext "subjectAltName=DNS:\${TLS_HOSTNAME},IP:\${TLS_PUBLIC_IP}" \\
  -addext "keyUsage=digitalSignature" \\
  -addext "extendedKeyUsage=serverAuth,clientAuth"
EOT

ENTRYPOINT ["/generate_tls_key.sh"]