# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM const-gen AS const-export
FROM docker.io/python:3.10@sha256:eeed7cac682f9274d183f8a7533ee1360a26acb3616aa712b2be7896f80d8c5f

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Debian
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

RUN wget https://github.com/algorand/go-algorand/releases/download/v3.19.0-stable/node_stable_linux-amd64_3.19.0.tar.gz
RUN tar --extract --file node_stable_linux-amd64_3.19.0.tar.gz bin/goal && rm node_stable_linux-amd64_3.19.0.tar.gz

RUN apt-get update
RUN apt-get -y install netcat

RUN mkdir teal
COPY Pipfile Pipfile.lock ./

RUN python3 -m pip install pipenv
RUN pipenv install

COPY *.py .
COPY test/*.json .
COPY deploy.sh deploy.sh 
COPY --from=const-export .env .env
