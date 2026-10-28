FROM docker.io/golang:1.19.13-bookworm@sha256:da9da58d86d106a5dda2ce249b00cf3b31cdd626ea41597e476de7b4eebad8c4

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat-openbsd
RUN apt install -y jq

WORKDIR /app

COPY ./wormchain/go.mod .
COPY ./wormchain/go.sum .
COPY ./sdk /sdk
RUN go mod download

# copy over c bindings (libwasmvm.x86_64.so, etc)
RUN cp -r /go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.1.1/internal/api/* /usr/lib

COPY ./wormchain .

EXPOSE 26657
EXPOSE 26656
EXPOSE 6060
EXPOSE 9090
EXPOSE 1317
EXPOSE 4500

RUN unset GOPATH

# create the dir for the default "home" config
RUN mkdir -p /root/.wormchain

ARG num_guardians
ENV NUM_GUARDIANS=$num_guardians

RUN /bin/bash /app/devnet/create-genesis.sh

RUN make client
RUN chmod +x /app/build/wormchaind


ENTRYPOINT ["/bin/bash","-c","/app/build/wormchaind start"]
