FROM docker.io/golang:1.21.0@sha256:b490ae1f0ece153648dd3c5d25be59a63f966b5f9e1311245c947de4506981aa

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat-openbsd
RUN apt install -y jq

WORKDIR /app

# Reference to: https://github.com/cosmos/relayer/commits/v2.4.2/
RUN curl -OL --ssl https://github.com/cosmos/relayer/archive/259b1278264180a2aefc2085f1b55753849c4815.tar.gz

RUN printf '%s %s\n' "1a3194dba53e8cc98e7e4c38d13c8c9f06c118aae69a38283eab1fd67d1348fe" "259b1278264180a2aefc2085f1b55753849c4815.tar.gz" | sha256sum --check
RUN tar xzf 259b1278264180a2aefc2085f1b55753849c4815.tar.gz
WORKDIR relayer-259b1278264180a2aefc2085f1b55753849c4815
RUN make install

COPY ./wormchain/ibc-relayer/chains /chains

RUN rly config init
RUN rly chains add-dir /chains
RUN rly keys restore localterra default "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius" --coin-type 330
RUN rly keys restore wormchain default "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius"
RUN rly paths new localterra wormchain terra-wormchain

EXPOSE 7597