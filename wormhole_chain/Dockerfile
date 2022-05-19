FROM docker.io/golang:1.17.5@sha256:90d1ab81f3d157ca649a9ff8d251691b810d95ea6023a03cdca139df58bca599

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat
RUN apt install -y jq
RUN curl https://get.starport.network/starport@v0.19.5! | bash

WORKDIR /app

COPY ./wormhole_chain/go.mod .
COPY ./wormhole_chain/go.sum .
COPY ./wormhole_chain/cosmos-sdk/go.mod .
COPY ./wormhole_chain/cosmos-sdk/go.sum .
RUN go mod download

COPY ./wormhole_chain .

EXPOSE 26657
EXPOSE 26656
EXPOSE 6060 
EXPOSE 9090 
EXPOSE 1317
EXPOSE 4500

RUN unset GOPATH

RUN make client
RUN chmod +x /app/build/wormhole-chaind
RUN make validators
RUN /app/build/wormhole-chaind collect-gentxs --home /app/build

ENTRYPOINT ["/bin/bash","-c","/app/build/wormhole-chaind start"]
