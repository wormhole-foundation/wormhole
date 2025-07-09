# Wormhole Guardian Node

The guardian node software is responsible for monitoring Wormhole connected chains, reporting on-chain observations,
aggregating those observations with those reported by other guardians until quorum is reached, and publishing signed VAAs.

For details on how to set up and run a guardian node see [operations.md](operations.md).

## Components

### Watchers

The watchers are responsible for monitoring the connected chains for Wormhole messages. They listen for messages published
by the Wormhole core contract and post them to the processor. They also handle requests to re-observe messages that may have
been missed. Additionally they monitor the latest block posted on chain, which is published in gossip heartbeat messages.

Each watcher listens to one chain, and there are a number of different watcher types:

1. EVM - the EVM watcher connects to one of the EVM chains using the `go-ethereum` library. It uses the EVM subscription
   mechanism to listen for new logs from the Wormhole core contract as well as the latest blocks. Additionally, it polls for
   new finalized and safe blocks (as supported on each chain).

2. Solana - the Solana watcher monitors SVM based chains, namely Solana, Pyth and Fogo. It uses the `solana-go` library.
   For each chain, there are two watchers, one listening for confirmed observations, and one listening for finalized observations.

   - For Solana, it polls for slots and searches for Wormhole transactions in all slots since the previous one.
   - For Pyth, it uses a web socket subscription to listen for messages.
   - For Fogo, it polls for any transactions involving the core contract since the previous one.

3. Cosmwasm - the Cosmwasm watcher connects to the various Cosmwasm chains (one chain per watcher instance). It subscribes
   for events from the Wormhole core contract and polls for new blocks.

4. Sui / Aptos / Algorand / Near - there are bespoke watchers for each of these chains that listen / poll for messages from
   the Wormhole core contract and poll for new blocks.

5. IBC - there is a Cosmwasm based watcher that listens to the IBC relayer contract on Gateway and publishes messages. The
   IBC watcher is different from the others in that a single watcher instance monitors _all_ IBC connected chains.

### P2P

The p2p package is responsible for listening to messages from and publishing messages to the libp2p based gossip network.
All of the guardians communicate over gossip. Most importantly, they publish observations they make and signed VAAs
for all observations that they observe reaching quorum. Additionally, they routinely publish heartbeats and various other
status events (such as from the governor).

The p2p package primarily posts events to and receives events from the processor package using golang channels.

The p2p package also maintains a separate pair of topics used for receiving and sending Wormhole Queries messages.
The guardian joins both of these topics, but it only subscribes to the request topic. This means a given guardian
does not see the Queries responses from other guardians. The guardian also applies libp2p filters so that it only
processes requests from a select set of peers.

### Processor

The processor takes messages observed from the watchers and observations gossipped by other guardians. It aggregates these
until an observation is seen by a quorum of guardians, including itself. It then publishes the signed VAA. The processor
stores signed VAAs in an on-disk badgerDB instance. The processor also interfaces with the governor and accountant packages
before publishing an observation.

### Governor

The governor is responsible for verifying that a given token bridge transfer will not exceed a daily limit for the given
chain and is not too large. For a detailed descriptions of the governor, see [governor.md](governor.md).

### Accountant

The accountant package is responsible for interfacing with the accountant contracts on Gateway to verify that a given
transfer will not exceed the available notional value for a chain. The accountant package interfaces with two contracts.
The accountant contract is used to monitor Token Bridge transfers and the NTT-accountant is used to monitor NTT transfers.

### Query Support

The query package is used to process Wormhole Queries (also known as CCQ, which stands for Cross Chain Queries) requests,
forwarding them to the appropriate watcher for processing, and posting the responses. It receives requests and publishes
responses over a separate set of P2P topics, via the p2p package.

Queries are currently supported on EVM and Solana.

### Admin Interface

The guardian supports a variety of admin commands to do things like request re-observation or sign a given payload using its key.

### Public RPC Endpoint

Certain guardians may be configured as a public RPC endpoint. If this feature is enabled, the guardian will listen for https requests
for things like the status of a given VAA.

For a list of the guardian public RPC endpoints see `PublicRPCEndpoints` in [sdk/mainnet_consts.go](../sdk/mainnet_consts.go)

### Guardian Key Management

Originally, the guardian signing key had to be stored in an armored file on local disk. The guardian signer feature added support
for other signing methods, including the existing armored file and Amazon KMS. The interface is generic, meaning that additional
signing techniques may be added going forward.

To enable a custom signer, use the `guardianSignerUri` config parameter.

### Alternate Publishing

The guardian supports the ability to publish observations to one or more configured HTTP endpoints. This is in addition to publishing
over P2P. This feature is described in more detail [here](../node/pkg/altpub/README.md).

### Transfer Verification

The guardian supports the ability to verify that transfers are valid. This feature is currently supported for certain EVM chains and Sui.

The verifier can be run as a stand alone monitoring tool as described [here](../node/cmd/txverifier/README.md). Or it can be enabled in the
guardian. In that case, suspect transfers will be blocked, meaning the observation will not be published. For more information on this mode,
see [here](../node/pkg/txverifier//README.md).

### Logs and Metrics

The guardian uses the [Zap logger](https://pkg.go.dev/go.uber.org/zap) for all of it's logging
and [Prometheus](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus) for metrics.
The logs are always written to local disk and the metrics are always available locally on the box.

The guardian can also be configured to publish logs to Grafana. To enable this, set the `telemetryLokiURL`
config parameter to point at a Grafana endpoint. In this mode, it still logs to local disk, but it
also publishes to Grafana in a non-blocking manner.

The guardian can also be configured to publish metrics to Grafana. To enable this, set the `promRemoteURL`
config parameter to point at a Grafana endpoint. In this mode, metrics will be posted to Grafana every
fifteen seconds, also in a non-blocking manner.

## Observation Lifecycle

An observation transitions through the following steps:

1. The watcher observes a message published by the Wormhole core contract.

2. On certain chains that do not have instant finality, the watcher waits until the block in which the message was
   published reaches the appropriate state, such as the block is finalized, safe, or the integrator requested instant
   finality.
3. When the watcher determines the block containing the message has reached the designated finality, it posts it to the
   processor via a golang channel.

4. The processor performs integrity checks on the message (via the governor and accountant). Either of these
   components may cause the observation to be delayed (the governor up to twenty-four hours, the accountant until
   a quorum of guardians report the observation).

5. Once the message clears the integrity checks, the processor signs the observation using the guardian key and
   posts it to an internal golang channel for batch publishing.

6. The processor batches observations, waiting no more than a second before publishing a batch. The batches are posted
   to the p2p package for publishing as a signed observation batch. The processor also allows for certain observations
   (specifically Pyth messages) to be published without batching. These are published immediately with a batch size of one.

7. The p2p package posts the signed observation batches to gossip.

8. Other guardians receive the observation. Each guardian aggregates the observation until it reaches quorum and
   it has made the observation itself.

9. Once an observation reaches quorum, the processor generates a VAA and posts it to the p2p package for publishing
   as a signed VAA with quorum.

10. The p2p package posts the signed VAA to gossip.

## Reobservation Requests

The guardian supports requesting that a missed observation be reobserved. These requests can originate from three sources:

1. From the guardian operator via an admin command.

2. From another guardian via gossip message.

3. Generated internally from the processor or accountant packages.

An observation request contains the chain ID and the transaction ID of the missed observation. The request is forwarded
to the watcher, based on the chain ID. The watcher queries for the transaction using the transaction ID and then publishes
the reobserved message to the processor. Note that there is a throttling mechanism in place to avoid requesting reobservation
of the same transaction too frequently.
