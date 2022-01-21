# syntax=docker/dockerfile:1.2
FROM docker.io/algorand/stable:3.2.1@sha256:0a87978492680fd98e2cc410f59f2bfd7fef979f035eb060685d56a6e916b5bd

RUN mkdir -p /setup
ADD template.json /setup/
ADD setup.sh /setup/
ADD setup.py /setup/
ADD teal/vaa-verify.teal /setup/
ADD teal/vaa-processor-clear.teal /setup/
ADD teal/vaa-processor-approval.teal /setup/

RUN ./goal network create -n sandnet -r /network -t /setup/template.json && echo rawr

RUN mkdir -p /network/Node/kmd-v0.5

ADD token /network/Node/algod.token
ADD token /network/Node/kmd-v0.5/kmd.token
ADD config.json /network/Node/config.json
ADD kmd_config.json /network/Node/kmd-v0.5/kmd_config.json

ENV ALGORAND_DATA=/network/Node

ADD Pipfile.lock /setup/
ADD Pipfile /setup/
RUN apt-get update
RUN apt-get install -y python3-pip
RUN pip install pipenv
