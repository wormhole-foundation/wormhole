# From https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/Dockerfile
FROM node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb AS tls

RUN apt-get update && apt-get -y install openssl

# Generate the TLS key and certificate
COPY <<EOT generate_tls_key.sh
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

RUN chmod +x generate_tls_key.sh

ENTRYPOINT ["./generate_tls_key.sh"]