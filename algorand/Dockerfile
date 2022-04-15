# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM docker.io/fedora:34 AS algorand-contracts

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Fedora
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/pki/tls/certs/ca-bundle.crt; fi

RUN dnf -y install python3-pip netcat wget

RUN dnf -y groupinstall "Development Tools" "Development Libraries"

COPY Pipfile.lock Pipfile.lock
COPY Pipfile Pipfile

RUN pip install pipenv
RUN pipenv install
RUN mkdir teal

COPY TmplSig.py TmplSig.py
COPY admin.py admin.py
COPY gentest.py gentest.py
COPY globals.py globals.py
COPY inlineasm.py inlineasm.py
COPY local_blob.py local_blob.py
COPY portal_core.py portal_core.py
COPY token_bridge.py token_bridge.py
COPY vaa_verify.py vaa_verify.py
COPY deploy.sh deploy.sh 


