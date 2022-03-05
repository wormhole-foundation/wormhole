# syntax=docker/dockerfile:1.2
FROM docker.io/algorand/testnet:latest

RUN mkdir -p /setup
ADD template.json /setup/
ADD setup.sh /setup/
ADD setup.py /setup/
#ADD teal/vaa-verify.teal /setup/
#ADD teal/vaa-processor-clear.teal /setup/
#ADD teal/vaa-processor-approval.teal /setup/

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
