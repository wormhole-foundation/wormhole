FROM docker.io/golang:1.19.0@sha256:4c00329e17be6fedd8bd4412df454a205348da00f9e0e5d763380a29eb096b75

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat
RUN apt install -y jq
RUN curl https://get.ignite.com/cli@v0.23.0 | bash && mv ignite /usr/local/bin/

WORKDIR /app

COPY ./wormhole_chain/go.mod .
COPY ./wormhole_chain/go.sum .
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
