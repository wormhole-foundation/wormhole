In order to run spy_relay successfully you need to do:

```
docker pull redis
```

The above will grab the docker for redis.
In order to run that docker use a command similar to:

```
docker run --rm -p6379:6379 --name redis-docker -d redis
```

To run the redis GUI do the following:

```
sudo apt-get install snapd
sudo snap install redis-desktop-manager
cd /var/lib/snapd/desktop/applications; ./redis-desktop-manager_rdm.desktop
```

To build the spy / guardian docker container:

```
cd node
docker build -f Dockerfile -t guardian .
```

To run the docker image in TestNet:

```
docker run -p 7073:7073 --entrypoint /guardiand guardian spy --nodeKey /node.key --spyRPC "[::]:7073" --network /wormhole/testnet/2/1 --bootstrap /dns4/wormhole-testnet-v2-bootstrap.certus.one/udp/8999/quic/p2p/12D3KooWBY9ty9CXLBXGQzMuqkziLntsVcyz4pk1zWaJRvJn6Mmt
```

To run spy_relay:

```
npm ci
npm run spy_relay
```

## Spy Listener Environment variables

see .env.tilt.listener for an example

- SPY_SERVICE_HOST - host & port string to connect to the spy
- SPY_SERVICE_FILTERS - Addresses to monitor (Bridge contract addresses) array of ["chainId","emitterAddress"]. Emitter addresses are native strings.
- REDIS_HOST - ip / host for the REDIS instance.
- REDIS_PORT - port number for redis.
- REST_PORT - port that the REST entrypoint will listen on.
- READINESS_PORT - port for kubernetes readiness probe
- LOG_LEVEL - log level, such as debug
- SUPPORTED_TOKENS - Origin assets that will attempt to be relayed. Array of ["chainId","address"], address should be a native string.

## Spy Relayer Environment variables

see .env.tilt.relayer for an example

- SUPPORTED_CHAINS - The configuration for each chain which will be relayed. See chainConfigs.example.json for the format. Of note, walletPrivateKey is an array, and a separate worker will be spun up for every private key provided.
- REDIS_HOST - host of the redis service, should be the same as in the spy_listener
- REDIS_PORT - port for redis to connect to
- PROM_PORT - port where prometheus monitoring will listen
- READINESS_PORT - port for kubernetes readiness probe
- CLEAR_REDIS_ON_INIT - boolean, if TRUE the relayer will clear the PENDING and WORKING Redis tables before it starts up.
- DEMOTE_WORKING_ON_INIT - boolean, if TRUE the relayer will move everything from the WORKING Redis table to the PENDING one.
- LOG_LEVEL - log level, debug or info
