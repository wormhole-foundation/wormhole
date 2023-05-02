FROM docker.io/golang:1.19.9@sha256:86901bd215ca2b33bdef68f70b999140252169bb55883bb29347334fa00b5b62

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat
RUN apt install -y jq

WORKDIR /app

RUN curl -OL --ssl https://github.com/cosmos/relayer/archive/c30b8d9e0ba6b45f8a576f768cad0777de545cb3.tar.gz
RUN printf '%s %s\n' "ede065997ee5873b6256748662fb93cd2652f45cb19937b7e6873ecc088109d9" "c30b8d9e0ba6b45f8a576f768cad0777de545cb3.tar.gz" | sha256sum --check
RUN tar xzf c30b8d9e0ba6b45f8a576f768cad0777de545cb3.tar.gz
WORKDIR relayer-c30b8d9e0ba6b45f8a576f768cad0777de545cb3
RUN make install

COPY ./wormchain/ibc-relayer/chains /chains

RUN rly config init
RUN rly chains add-dir /chains
RUN rly keys restore localterra default "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius" --coin-type 330
RUN rly keys restore wormchain default "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius"
RUN rly paths new localterra wormchain terra-wormchain

EXPOSE 7597