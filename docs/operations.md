# Running a Wormhole Guardian Node

![](images/nodearchitecture.svg)

## Connected chains

In addition to Wormhole itself, you need to run your own verifying node for every chain that Wormhole connects to except
for newer IBC connected chains that integrate through Wormhole Gateway. Please refer to the [constants reference](https://docs.wormhole.com/wormhole/reference/constants)
for all chains that Wormhole connects to.

**Do NOT use third-party RPC service providers** for any of the chains! You'd fully trust them, and they could lie to
you on whether an event has actually been observed. The whole point of Wormhole is not to rely on centralized nodes!

We strongly recommend running your own full nodes for both testnet and mainnet (where applicable)
so you can test changes for your mainnet full nodes and gain operational experience.

### Solana node requirements

Refer to the [Solana documentation](https://docs.solanalabs.com/operations/setup-an-rpc-node) on how to run an RPC
(full) node.  [Solana's Discord server](https://solana.com/community) is a great resource for questions regarding
operations.

The `#rpc-server-operators` channel is especially useful for setting up Solana RPC nodes.

Your Solana RPC node needs the following parameters enabled:

```
--enable-rpc-transaction-history
--enable-cpi-and-log-storage
```

`--enable-rpc-transaction-history` enables historic transactions to be retrieved via the _getConfirmedBlock_ API,
which is required for Wormhole to find transactions.

`--enable-cpi-and-log-storage` stores metadata about CPI calls.

Be aware that these require extra disk space!

#### Account index

If you use the same RPC node for Wormhole v1, you also need the following additional parameters to speed up
`getProgramAccount` queries:

<!-- cspell:disable -->

```
[... see above for other required parameters ...]

--account-index program-id
--account-index-include-key WormT3McKhFJ2RkiGpdw9GKvNCrB2aB54gb2uV9MfQC   # for mainnet
--account-index-include-key 5gQf5AUhAgWYgUCt9ouShm9H7dzzXUsLdssYwe5krKhg  # for testnet
```

<!-- cspell:enable -->

Alternatively, if you want to run a general-purpose RPC node with indexes for all programs instead of only Wormhole,
leave out the filtering:

```
--account-index program-id
```

On mainnet, we strongly recommend blacklisting KIN and the token program to speed up catchup:

<!-- cspell:disable -->

```
--account-index-exclude-key kinXdEcpDQeHPEuQnqmUgtYykqKGVFq6CeVX5iAHJq6  # Mainnet only
--account-index-exclude-key TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA  # Mainnet only
```

<!-- cspell:enable -->

Note that these indexes require extra disk space and may slow down catchup. The first startup after
adding these parameters will be slow since Solana needs to recreate all indexes.

### Ethereum node requirements

In order to observe events on the Ethereum chain, you need access to an Ethereum RPC endpoint. The most common
choice is geth, but for the sake of diversity, you may want to run something that isn't geth.

With RPC providers such as Alchemy, Infura, etc. you trust those operators to provide you with untampered chain data and
have no way of verifying correctness. Therefore, Wormhole requires either an Ethereum full-node or a light-client. The
node can be operated in the full, quick or light modes with no impact on security or performance of the bridge software.
As long as the node supports the Ethereum JSON RPC API, it will be compatible with the bridge so all major
implementations will work fine.

Generally, full-nodes will work better and be more reliable than light clients which are susceptible to DoS attacks
since only very few nodes support the light client protocol.

Running a full node typically requires ~500G of SSD storage, 8G of RAM and 4-8 CPU threads (depending on clock
frequency). Light clients have much lower hardware requirements.


### Terra

Terra is also sometimes referred to as Terra 2, but the correct name is just simply "Terra". The previous version was renamed to "Terra Classic".

Refer to the [Terra documentation](https://docs.terra.money/full-node/run-a-full-terra-node/set-up-production/) on how to run a full node.

#### Terra Classic

Refer to the [Terra Classic documentation](https://classic-docs.terra.money/docs/full-node/run-a-full-terra-node/README.html) on how to run a full node.


### Wormchain

All guardians **must run validators for wormchain**, the codename of [Wormhole Gateway](https://wormhole.com/gateway/).

The ``--wormchainURL` argument to the guardian node should point to `<validator address>:9090` which is the `grpc` port
in the app.toml.

Example port setup:

<!-- cspell:disable -->

config.toml:

```toml
[rpc]
laddr = "tcp://0.0.0.0:26657"
grpc_laddr = ""
pprof_laddr = "localhost:6060"

[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = ""
```

app.toml:

```toml
[grpc]
address = "0.0.0.0:9090"

[grpc-web]
address = "0.0.0.0:9091"
```

<!-- cspell:enable -->

For signing, consider setting up a remote threshold signer such as
[horcrux](https://github.com/strangelove-ventures/horcrux) and adopting the sentry node architecture with sentry nodes
in front of your wormchain validator.

#### Wormchain Useful Commands

Check the latest guardian set:

<!-- cspell:disable -->

```shell
$ wormchaind query wormhole latest-guardian-set-index
latestGuardianSetIndex: 4
```

<!-- cspell:enable -->

Upgrade the guardian set (with a valid governance vaa):

<!-- cspell:disable -->

```shell
wormchaind tx wormhole execute-governance-vaa <guardian_set_upgrade_VAA_in_hex_format>
```

<!-- cspell:enable -->

View Validator information:

<!-- cspell:disable -->

```shell
$ wormchaind q staking validators
... snip ...
- commission:
    commission_rates:
      max_change_rate: "0.020000000000000000"
      max_rate: "0.200000000000000000"
      rate: "0.000000000000000000"
    update_time: "2024-04-16T19:13:45.210176030Z"
  consensus_pubkey:
    '@type': /cosmos.crypto.ed25519.PubKey
    key: T+hsVX52EarrsL+mOwv3mL0byWa2EctsG6XmikUMFiQ=
  delegator_shares: "0.000000000000000000"
  description:
    details: ""
    identity: 11A4103C4BCBD2B4
    moniker: RockawayX
    security_contact: ""
    website: https://rockawayx.com/infrastructure
  jailed: false
  min_self_delegation: "0"
  operator_address: wormholevaloper1thl5syhmscgnj7whdyrydw3w6vy80044278fxp
  status: BOND_STATUS_BONDED
  tokens: "0"
  unbonding_height: "0"
  unbonding_time: "1970-01-01T00:00:00Z"
```

<!-- cspell:enable -->

### EVM node requirements

Some non-Ethereum EVM compatible blockchains need to run in archive mode for [Queries](https://wormhole.com/queries)
to function correctly. By default in geth, [historical state is only kept in memory for the previous 128 blocks](https://github.com/ethereum/go-ethereum/blob/4458905f261d5d9ba5fda3d664f9bb80346ab404/core/state/statedb.go#L1259-L1265).
After 128 blocks, older states are garbage collected. Many of these chains are forks of geth that maintain this
historical limitation.

* Arbitrum
* Base
* Optimism

Newer execution clients such as [reth](https://github.com/paradigmxyz/reth) lack this limitation and are worth
investigating once they are stable.

Additionally, if there is ever a scenario where the network fails to come to consensus on an EVM compatible chain due to
a hard fork or some unforeseen scenario, it might be required to run archive nodes for those chains temporarily to ensure
the transactions can be reobserved.

### Cosmos / IBC connected nodes

All modern Cosmos integrations happen by Wormhole observing IBC transactions on Gateway (wormchain). Guardian node operators do not need to run full nodes for these networks. For Cosmos based chains that were added before this functionality, a full node is still necessary.

The following Cosmos based nodes were added prior to Gateway and guardians need to run full nodes:

* Injective
* Terra
* Terra Classic
* XPLA

**NOTE**: All guardians must run validators for wormchain.

## Building guardiand

For security reasons, we do not provide a pre-built binary. You need to check out the repo and build the
guardiand binary from source. A Git repo is much harder to tamper with than release binaries.

To build the Wormhole node, you need [Go](https://golang.org/dl/) >= 1.21.9

First, check out the version of the Wormhole repo that you want to deploy:

```bash
git clone https://github.com/wormhole-foundation/wormhole && cd wormhole
```

Then, compile the release binary as an unprivileged build user:

```bash
make node
```

You'll end up with a `guardiand` binary in `build/`.

Consider these recommendations, not a tutorial to be followed blindly. You'll want to integrate this with your
existing build pipeline. If you need Dockerfile examples, you can take a look at our devnet deployment.

If you want to compile and deploy locally, you can run `sudo make install` to install the binary to /usr/local/bin.

If you deploy using a custom pipeline, you need to set the `CAP_IPC_LOCK` capability on the binary (e.g. doing the
equivalent to `sudo setcap cap_ipc_lock=+ep`) to allow it to lock its memory pages to prevent them from being paged out.
See below on why - this is a generic defense-in-depth mitigation which ensures that process memory is never swapped out
to disk. Please create a GitHub issue if this extra capability represents an operational or compliance concern.

## Key Generation

To generate a guardian key, install guardiand first. If you generate the key on a separate machine, you may want to
compile guardiand only without installing it:

    make node
    sudo setcap cap_ipc_lock=+ep ./build/bin/guardiand

Otherwise, use the same guardiand binary that you compiled using the regular instructions above.

Generate a new key using the `keygen` subcommand:

    guardiand keygen --desc "Testnet key foo" /path/to/your.key

The key file includes a human-readable part which includes the public key hashes and the description.

## Deploying

We strongly recommend a separate user and systemd services for the Wormhole services.

See the separate [wormhole-networks](https://github.com/wormhole-foundation/wormhole-networks) repository for examples
on how to set up the guardiand unit for a specific network.

You need to open port 8999/udp in your firewall for the P2P network and 8996/udp for
[Cross Chain Queries](../whitepapers/0013_ccq.md). Nothing else has to be exposed externally if you do not run a public RPC.

journalctl can show guardiand's colored output using the `-a` flag for binary output, i.e.: `journalctl -a -f -u guardiand`.

### Kubernetes

Kubernetes deployment is fully supported.

Refer to [devnet/](../devnet) for example k8s deployments as a starting point for your own production deployment. You'll
have to build your own containers. Unless you already run Kubernetes in production, we strongly recommend a traditional
deployment on a dedicated instance - it's easier to understand and troubleshoot.

When running in kubernetes, or behind any kind of NAT, pass `--gossipAdvertiseAddress=external.ip.address` to the
guardiand node process to ensure the external address is advertized in p2p. If this is not done, reobservation
requests and [CCQ](https://wormhole.com/queries) will not function as intended.

### Monitoring

Wormhole exposes a status server for readiness and metrics. By default, it listens on port 6060 on localhost.
You can use a command line argument to expose it publicly: `--statusAddr=[::]:6060`.

**NOTE:** Parsing the log output for monitoring is NOT recommended. Log output is meant for human consumption and is
not considered a stable API. Log messages may be added, modified or removed without notice. Use the metrics :-)

#### `/readyz`

This endpoint returns a 200 OK status code once the Wormhole node is ready to serve requests. A node is
considered ready as soon as it has successfully connected to all chains and started processing requests.

This is **only for startup signaling** - it will not tell whether it _stopped_
processing requests at some later point. Once it's true, it stays true! Use metrics to figure that out.

#### `/metrics`

This endpoint serves [Prometheus metrics](https://prometheus.io/docs/concepts/data_model/) for alerting and
introspection. We recommend using Prometheus and Alertmanager, but any monitoring tool that can ingest metrics using the
standardized Prometheus exposition format will work.

Once we gained more operational experience with Wormhole, specific recommendations on appropriate symptoms-based
alerting will be documented here.

See [Wormhole.json](../dashboards/Wormhole.json) for an example Grafana dashboard.

#### Wormhole Dashboard

There is a [dashboard](https://wormhole-foundation.github.io/wormhole-dashboard) which shows the overall health of the
network and has metrics on individual guardians.

**NOTE:** Parsing the log output for monitoring is NOT recommended. Log output is meant for human consumption and is
not considered a stable API. Log messages may be added, modified or removed without notice. Use the metrics :-)

#### Wormhole Fly Healthcheck

In the [wormhole-dashboard](https://github.com/wormhole-foundation/wormhole-dashboard) repository, there is a small
[healthcheck application](https://github.com/wormhole-foundation/wormhole-dashboard/tree/main/fly/cmd/healthcheck)
which verifies that the guardian is gossiping out heartbeats, is submitting chain observations, and has a working
heartbeats API available. This is a very good way to verify a specific guardian is functioning as intended.

You can clone the repo and run the check against the [MCF Guardian](https://github.com/wormhole-foundation/wormhole-networks/blob/649dcc48f29d462fe6cb0062cb6530021d36a417/mainnetv2/guardianset/v3.prototxt#L58):

```shell
git clone https://github.com/wormhole-foundation/wormhole-dashboard
cd wormhole-dashboard/fly/cmd/healthcheck

# Run the fly
$ go run main.go --pubKey 0xDA798F6896A3331F64b48c12D1D57Fd9cbe70811 --url https://wormhole-v2-mainnet-api.mcf.rocks
✅ guardian heartbeat received {12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU: [/ip4/185.188.42.109/udp/8999/quic-v1]}
✅ 44 observations received
✅ /v1/heartbeats
```

If the guardian public RPC is not exposed, the `--url` flag can be omitted:

```shell
$ go run main.go --pubKey 0xDA798F6896A3331F64b48c12D1D57Fd9cbe70811
✅ guardian heartbeat received {12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU: [/ip4/185.188.42.109/udp/8999/quic-v1]}
✅ 41 observations received
ℹ️  --url not defined, skipping web checks
```

The bootstrap nodes and network defaults to mainnet and the values can be found in the [network constants](../node/pkg/p2p/network_consts.go).

It can also be used to test a specific bootstrap node/s:

```shell
$ go run main.go --pubKey 0xDA798F6896A3331F64b48c12D1D57Fd9cbe70811 --bootstrap /dns4/wormhole.mcf.rocks/udp/8999/quic/p2p/12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU
✅ guardian heartbeat received {12D3KooWDZVv7BhZ8yFLkarNdaSWaB43D6UbQwExJ8nnGAEmfHcU: [/ip4/185.188.42.109/udp/8999/quic-v1]}
✅ 44 observations received
ℹ️  --url not defined, skipping web checks
```

## Native Token Transfers

[NTT](https://github.com/wormhole-foundation/example-native-token-transfers) is an exciting feature of wormhole that builds upon the core bridge to allow mint/burn style transfers. Ensuring it runs correctly requires integrating it with the NTT Accountant. To enable this feature, create a **new** wormchain key. Do not reuse an existing global accountant key and add the following parameters:

<!-- cspell:disable -->

```shell
# You may already have these.
--wormchainURL URL_TO_YOUR_WORMCHAIN_NODE
--accountantWS HTTP_URL_OF_YOUR_WORMCHAIN_NODE

# This is the mainnet contract.
--accountantNttContract wormhole1mc23vtzxh46e63vq22e8cnv23an06akvkqws04kghkrxrauzpgwq2hmwm7

--accountantNttKeyPath PATH_TO_YOUR_NTT_ACCOUNTANT_KEY_FILE
--accountantNttKeyPassPhrase YOUR_NTT_ACCOUNTANT_KEY_PASS_PHRASE
```

<!-- cspell:enable -->

Please remember to allowlist the new NTT Accountant key for use with Wormchain! For instructions on how to do that, speak with someone from the Wormhole Foundation.

## Cross-Chain Queries

[CCQ](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0013_ccq.md) also known as [Wormhole Queries](https://wormhole.com/queries) is a feature to allow pulling attestations in a cross chain manner. To run ccq, a few additional flags need to be enabled on the guardian node:

<!-- cspell:disable -->

```shell
--ccqEnabled=true \
--ccqAllowedPeers="[ALLOWED,PEERS,GO,HERE]" \
--ccqAllowedRequesters="[ALLOWED,REQUESTORS,GO,HERE" \
```

<!-- cspell:enable -->

To test query functionality, follow the instructions in [node/hack/query/ccqlistener/ccqlistener.go](../node/hack/query/ccqlistener/ccqlistener.go).

## Running a public API endpoint

Wormhole v2 no longer uses Solana as a data availability layer (see [design document](../whitepapers/0005_data_availability.md)).
Instead, it relies on Guardian nodes exposing an API which web wallets and other clients can use to retrieve the signed VAA
message for a given transaction.

Guardian nodes are **strongly encouraged** to expose a public API endpoint to improve the protocol's robustness.

guardiand comes with a built-in REST and grpc-web server which can be enabled using the `--publicWeb` flag:

```
--publicWeb=[::]:443
```

For usage with web wallets, TLS needs to be supported. guardiand has built-in Let's Encrypt support:

```
--tlsHostname=wormhole-v2-mainnet-api.example.com
--tlsProdEnv=true
```

Alternatively, you can use a managed reverse proxy like CloudFlare to terminate TLS.

It is safe to expose the publicWeb port on signing nodes. For better resiliency against denial of service attacks,
future guardiand releases will include listen-only mode such that multiple guardiand instances without guardian keys
can be operated behind a load balancer.

## Enabling Telemetry

Optionally, the guardian can send telemetry to [Grafana Cloud Logs](https://grafana.com/products/cloud/logs/) aka "loki".
To enable this functionality, add the following flag:

```bash
--telemetryLokiURL=$PER_GUARDIAN_LOKI_URL_WITH_TOKEN
```

New guardians should talk to the Wormhole Foundation to get a Loki url.

### Binding to privileged ports

If you want to bind `--publicWeb` to a port <1024, you need to assign the CAP_NET_BIND_SERVICE capability.
This can be accomplished by either adding the capability to the binary (like in non-systemd environments):

     sudo setcap cap_net_bind_service=+ep guardiand

...or by extending the capability set in `guardiand.service`:

    AmbientCapabilities=CAP_IPC_LOCK CAP_NET_BIND_SERVICE
    CapabilityBoundingSet=CAP_IPC_LOCK CAP_NET_BIND_SERVICE

## Key Management

You'll have to manage the following keys:

- The **guardian key**, which is the bridge consensus key. This key is very critical - your node uses it to certify
  VAA messages. The public key's hash is stored in the guardian set on all connected chains. It does not accrue rewards.
  It's your share of the multisig mechanism that protect the Wormhole network. The guardian set can be replaced
  if a majority of the guardians agree to sign and publish a new guardian set.

- A **node key**, which identifies it on the gossip network, similar to Solana's node identity or a Tendermint
  node key. It is used by the peer-to-peer network for routing and transport layer encryption.
  An attacker could potentially use it to censor your messages on the network. Other than that, it's not very
  critical and can be rotated. The node will automatically create a node key at the path you specify if it doesn't exist.
  While the node key can be replaced, we recommend using a persistent node key. This will make it easier to identify your
  node in monitoring data and improves p2p connectivity.

For production, we strongly recommend to either encrypt your disks, and/or take care to never have hot guardian keys touch the disk.
One way to accomplish is to store keys on an in-memory ramfs, which can't be swapped out, and restore it from cold
storage or an HSM/vault whenever the node is rebooted. You might want to disable swap altogether. None of that is
specific to Wormhole - this applies to any hot keys.

Our node software takes extra care to lock memory using mlock(2) to prevent keys from being swapped out to disk, which
is why it requires extra capabilities. Yes, other chains might want to do this too :-)

Storing keys on an HSM or using remote signers only partially mitigates the risk of server compromise - it means the key
can't get stolen, but an attacker could still cause the HSM to sign malicious payloads. Future iterations of Wormhole
may include support for remote signing.

## Bootstrap Peers

The list of supported bootstrap peers is defined in [node/pkg/p2p/network_consts.go](../node/pkg/p2p/network_consts.go).
That file also provides golang functions for obtaining the network parameters (network ID and bootstrap peers) based on
the environment (mainnet or testnet).

The common Wormhole applications (guardiand, spy and query proxy server) use those functions, so it is not necessary to specify
the actual bootstrap parameters in their configs. Developers of any new applications are strongly urged to do the same, and not
proliferate lists of bootstrap peers which might change over time.

## Run the Guardian Spy

The spy connects to the wormhole guardian peer to peer network and listens for new VAAs. It publishes those via a socket and websocket that applications can subscribe to. If you want to run the spy built from source, change `ghcr.io/wormhole-foundation/guardiand:latest` to `guardian` after building the `guardian` image.

Start the spy against the testnet wormhole guardian:

<!-- cspell:disable -->

```bash
docker run \
    --pull=always \
    --platform=linux/amd64 \
    -p 7073:7073 \
    --entrypoint /guardiand \
    ghcr.io/wormhole-foundation/guardiand:latest \
    spy --nodeKey /node.key --spyRPC "[::]:7073" --env testnet
```

<!-- cspell:enable -->

To run the spy against mainnet:

<!-- cspell:disable -->

```bash
docker run \
    --pull=always \
    --platform=linux/amd64 \
    -p 7073:7073 \
    --entrypoint /guardiand \
    ghcr.io/wormhole-foundation/guardiand:latest \
    spy --nodeKey /node.key --spyRPC "[::]:7073" --env mainnet
```

<!-- cspell:enable -->

## Guardian Configurations

Configuration files, environment variables and flags are all supported.

### Config File

**Location/Naming**: By default, the config file is expected to be in the `node/config` directory. The standard name for the config file is `guardiand.yaml`. Currently there's no support for custom directory or filename yet.

**Format**: We support any format that is supported by [Viper](https://pkg.go.dev/github.com/dvln/viper#section-readme). But YAML format is generally preferred.

**Example**:

<!-- cspell:disable -->

```yaml
ethRPC: "ws://eth-devnet:8545"
ethContract: "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
solanaRPC: "http://solana-devnet:8899"
solanaContract: "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
```

<!-- cspell:enable -->

### Environment Variables

**Prefix**: All environment variables related to the Guardian node should be prefixed with `GUARDIAND_`.

**Usage**: Environment variables can be used to override settings in the config file. Particularly for sensitive data like API keys that should not be stored in config files.

**Example**:

```bash
export GUARDIAND_ETHRPC=ws://eth-devnet:8545
```

### Command-Line Flags

**Usage**: Flags provide the highest precedence and can be used for temporary overrides or for settings that change frequently.

**Example**:

```bash
./guardiand node --ethRPC=ws://eth-devnet:8545
```

### Precedence Order

The configuration settings are applied in the following order of precedence:

1. **Command-Line Flags**: Highest precedence, overrides any other settings.
2. **Environment Variables**: Overrides the config file settings but can be overridden by flags.
3. **Config File**: Lowest precedence.
