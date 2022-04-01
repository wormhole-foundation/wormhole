FROM docker.io/golang:1.17.5@sha256:90d1ab81f3d157ca649a9ff8d251691b810d95ea6023a03cdca139df58bca599

#used for a readiness probe
RUN apt-get update
RUN apt install -y netcat

WORKDIR /app

COPY ./wormhole_chain .

RUN curl https://get.starport.network/starport! | bash

EXPOSE 26657
EXPOSE 26656
EXPOSE 6060 
EXPOSE 9090 
EXPOSE 1317
EXPOSE 4500

RUN unset GOPATH

#This command initializes the config directory, using starport to generate the genesis file.
#This means the genesis will pick up the custom arguments from config.yml
RUN starport chain init --home /app/validators/guardian_validator
#This command builds the executable for our node, and places it into the directory we are building up, guardian_validator
#Building through starport ensures that we have the correct code from both the cosmos-sdk hardfork, and our custom modules.
RUN starport chain build --output /app/validators/guardian_validator
#This command takes the gentx from first_validator, and puts it into guardian_validator's genesis.json.
#This means that the gentx is deterministic across runs, and that first_validator is always the only validator registered at startup.
RUN chmod +x /app/validators/guardian_validator/wormhole-chaind
RUN /app/validators/guardian_validator/wormhole-chaind collect-gentxs --home /app/validators/guardian_validator --gentx-dir /app/validators/first_validator/genesis/

ENTRYPOINT ["/bin/bash","-c","/app/validators/guardian_validator/wormhole-chaind start"]
