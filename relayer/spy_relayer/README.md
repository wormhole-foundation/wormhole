Relayer
=======

The wormhole relayer is designed to answer one main question:

    Q: How do you transfer to a new wallet in a destination chain when you lack native tokens for paying gas fees?
    A: You pay a little bit more on the source chain and use that to pay gas fees on the destination chain.

It was originally designed for payload version 1 token transfers, but should be extensible to other payload types as well.


Architecture
------------

|   Component    |                                     Description                                    |
|----------------|------------------------------------------------------------------------------------|
| Guardian Spy   | Connects to the wormhole p2p network and publishes all VAAs to a websocket         |
| Spy Listener   | Filters VAAs from the Spy and adds them to the incoming queue in Redis             |
| REST Listener  | Accepts HTTP requests to relay VAAs and writes them to the incoming queue in Redis |
|     Redis      | A durable queue for storing VAAs before they are relayed                           |
|    Relayer     | Scans the Redis incoming queue and moves acceptable VAAs to the working queue. It then completes the transfer and pays gas fees on the destination chain. |
| Wallet Monitor | Presents a prometheus endpoint for monitoring wallet balances of native tokens (for paying gas fees) and non-native tokens as relayer profit |

If Redis is temporarily down, the Listener will queue outstanding transactions in memory. When Redis comes back online, the Listener writes them all to Redis.


### Architecture Diagram

This is a rough diagram of how the components fit together:

    ┌────────────────────────────────────────┐
    │ Wormhole Guardian Peer to Peer Network │
    └───────────────────┬────────────────────┘
                        │
                 ┌──────▼───────┐
                 │ Guardian Spy │
                 └──────┬───────┘
                        │
                 ┌──────▼───────┐
                 │ Spy Listener │
                 └──────┬───────┘
                        │
                    ┌───▼───┐    ┌───────────────┐
                    │ Redis │◄───┤ REST Listener │
                    └───┬───┘    └───────────────┘
                        │
                   ┌────▼────┐
                   │ Relayer │
                   └─────────┘
                        │
               ┌────────▼───────┐
               │ Wallet Monitor │
               └────────────────┘



Environment Variables
---------------------

### Listener

These are for configuring the spy and rest listener. See [.env.tilt.listener](.env.tilt.listener) for examples:

| Name | Description |
|------|-------------|
| `SPY_SERVICE_HOST` | host & port string to connect to the spy |
| `SPY_SERVICE_FILTERS` | Addresses to monitor (Wormhole core bridge contract addresses) array of ["chainId","emitterAddress"]. Emitter addresses are native strings. |
| `REDIS_HOST` | Redis host / ip to connect to |
| `REDIS_PORT` | Redis port |
| `REST_PORT` | Rest listener port to listen on. |
| `READINESS_PORT` | Kubernetes readiness probe port to listen on. |
| `LOG_LEVEL` | log level, such as debug |
| `SUPPORTED_TOKENS` | Origin assets that will attempt to be relayed. Array of ["chainId","address"], address should be a native string. |


### Relayer

These are for configuring the actual relayer. See [.env.tilt.relayer](.env.tilt.relayer) for examples:

| Name | Description |
|------|-------------|
| `SUPPORTED_CHAINS` | The configuration for each chain which will be relayed. See [chainConfigs.example.json](src/chainConfigs.example.json) for the format. Of note, `walletPrivateKey` is an array, and a separate worker will be spun up for every private key provided. |
| `REDIS_HOST` | host of the redis service, should be the same as in the spy_listener |
| `REDIS_PORT` | port for redis to connect to |
| `PROM_PORT` | port where prometheus monitoring will listen |
| `READINESS_PORT` | port for kubernetes readiness probe |
| `CLEAR_REDIS_ON_INIT` | boolean, if `true` the relayer will clear the INCOMING and WORKING Redis tables before it starts up. |
| `DEMOTE_WORKING_ON_INIT` | boolean, if `true` the relayer will move everything from the WORKING Redis table to the INCOMING one. |
| `LOG_LEVEL` | log level, debug or info |


Building
--------


### Building the Spy

To build the guardiand / spy container from source:

```bash
cd node
docker build -f Dockerfile -t guardian .
```

### Building the Relayer application

Build the relayer for non-containerized testing:

```bash
cd relayer/spy_relayer
npm ci
npm run build
```


Running the Whole Stack For Testing
-----------------------------------

This config is mostly for development.


### Run Redis

Start a redis container:

```bash
docker run --rm -p6379:6379 --name redis-docker -d redis
```

### Run the Guardian Spy

The spy connects to the wormhole guardian peer to peer network and listens for new VAAs. It publishes those via a socket and websocket that the listener subscribes to. If you want to run the spy built from source, change `ghcr.io/certusone/guardiand:latest` to `guardian` after building the `guardian` image.

Start the spy against the testnet wormhole guardian:

```bash
docker run \
    --platform=linux/amd64 \
    -p 7073:7073 \
    --entrypoint /guardiand \
    ghcr.io/certusone/guardiand:latest \
spy --nodeKey /node.key --spyRPC "[::]:7073" --network /wormhole/testnet/2/1 --bootstrap /dns4/wormhole-testnet-v2-bootstrap.certus.one/udp/8999/quic/p2p/12D3KooWBY9ty9CXLBXGQzMuqkziLntsVcyz4pk1zWaJRvJn6Mmt
```

### Run The Apps

This runs the Spy Listener, REST Listener, Relayer, and Wallet Monitor all in a single process for development and testing purposes:

Start the application:

```bash
npm ci
npm run spy_relay
```
