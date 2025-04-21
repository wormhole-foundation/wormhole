# Operating the Wormhole Queries Proxy Server

The Wormhole queries proxy server (sometimes referred to as the CCQ proxy) is a server that listens on a
REST endpoint for Wormhole query requests. It validates those requests and forwards them to the guardian
CCQ P2P network for processing by the guardians. It then accumulates the responses from the guardians,
verifies quorum and forwards the response to the client.

## Building the Proxy Server

The proxy server runs as another instance of the `guardiand` process, similar to the spy. It is built exactly
the same as the spy, and requires the same dependencies. Please see the [Operations Guide](operations.md#building-guardiand) for
details on how to build `guardiand`.

## Deploying the Proxy Server

The proxy server can be deployed just like the spy, including potentially running in a container. Note that it
requires a public IP address to listen for REST requests, and it needs to be able to reach the guardian P2P network.

The proxy is not particularly resource intensive, so should run successfully on a reasonable size VM.

## Configuring the Proxy Server

The proxy server is configured using command line arguments. The main configuration involves setting up
staking-based rate limiting, which uses on-chain staking information to determine access and rate limits
for query requests.

### Proxy Server Command Line Arguments

The following is a sample command line for running the proxy server in mainnet.

```shell
wormhole $build/bin/guardiand query-server \
      --env "mainnet" \
      --nodeKey /home/ccq/data/ccq_server.nodeKey \
      --signerKey "/home/ccq/data/ccq_server.signerKey" \
      --listenAddr "[::]:8080" \
      --ethRPC https://eth.drpc.org \
      --ethContract "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B" \
      --stakingPoolAddresses "0xPoolAddress1,0xPoolAddress2,0xPoolAddress3" \
      --ipfsGateway "https://ipfs.io" \
      --policyCacheDuration 300 \
      --logLevel=info \
      --telemetryLokiURL $LOKI_URL \
      --telemetryNodeName "Mainnet CCQ server 1" \
      --promRemoteURL $PROM_URL
```

- The `env` can be mainnet, testnet or devnet.
- The `nodeKey` should point to the file containing the P2P key. The first time the proxy runs, if the
  file does not exist, it will be created. You can look in the proxy server logs to get the generated key.
- The `signerKey` should point to an armored file containing a key that will be used to sign the server's
  responses. Please see below for how to generate this file.
- The `listenAddr` specifies the port on which the proxy listens for REST requests.
- The `ethRPC` and `ethContract` are used to read the wormhole guardian set on start up. The address
  above is for mainnet. If you are running in testnet, you should point to Holesky and use `0xa10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a`.
  (You can confirm these addresses [here](https://docs.wormhole.com/wormhole/reference/constants#contract-addresses).)
  Note that using a public endpoint should be fine, since the proxy only does a single read of the guardian set.
- The `stakingPoolAddresses` specifies a comma-separated list of staking pool contract addresses
  (e.g., "0xAddress1,0xAddress2,0xAddress3"). When provided, the proxy queries these pools directly
  for staking policies. **This is the preferred method for connecting to staking pools.** Either this
  parameter or `ccqFactoryAddress` must be provided to enable staking-based rate limiting.
- The `ccqFactoryAddress` specifies the address of the CCQ staking factory contract for factory-based
  pool discovery. This is an alternative to `stakingPoolAddresses` and should not be used together with it.
  **Note:** This only works with staking factory contracts deployed prior to commit
  [185e68f2](https://github.com/wormhole-foundation/queries-staking/commit/185e68f2295bda505b6eb427c59ff209c3555bd8).
  For newer deployments, you must use `stakingPoolAddresses`.
- The `ipfsGateway` specifies the IPFS gateway URL for fetching conversion tables (default: "https://ipfs.io").
- The `policyCacheDuration` specifies how long (in seconds) to cache staking policies (default: 300 = 5 minutes).
- The `telemetryLokiURL`, `telemetryNodeName` and `promRemoteURL` are used for telemetry purposes and
  the values will be provided by Wormhole Foundation personnel if appropriate.

Optional Parameters

- The `gossipAdvertiseAddress` argument allows you to specify an external IP to advertize on P2P (use if behind a NAT or running in k8s).
- The `monitorPeers` flag will cause the proxy server to periodically check its connectivity to the P2P bootstrap peers, and attempt to reconnect if necessary.

#### Creating the Signing Key File

Do the following to create the signing key file. Note that the `block-type` must exactly match what is specified below,
but the `desc` can be anything you want.

```shell
wormhole$ build/bin/guardiand keygen --desc "Your CCQ proxy server" --block-type "CCQ SERVER SIGNING KEY" /home/ccq/data/ccq_server.signerKey
```

### Guardian Support for a New Proxy Server

The Queries P2P network is permissioned. The guardians will ignore P2P traffic from sources that are not in their configuration.
Additionally, they will only honor query requests signed using a key in their configured list. Before you can begin publishing
requests from your proxy, you must get a quorum (preferably all) of the guardians to add your values for the following to their
configurations:

- P2P key (the value from `nodeKey` file, logged on proxy start up).
- The public key associated with the signing key. See the `signerKey` file.

Please work with foundation personnel to get your proxy server added to the guardian configurations.

### Staking-Based Rate Limiting

The proxy server uses a staking-based rate limiting system to control access to CCQ queries. Instead of
API keys and permissions files, access and rate limits are determined by on-chain staking.

#### How It Works

1. **Staking Requirement**: Users must have staked tokens in the CCQ staking factory contract to access the service.
2. **Rate Limits from Stake**: The amount of stake determines the rate limits for different query types.
3. **Signature-Based Authentication**: All requests must be signed by the user's wallet to prove ownership.
4. **Delegation Support**: Users can delegate their rate limits to other addresses by including a `StakerAddress`
   field in their query requests.

#### Request Authentication

Requests must be signed using ECDSA signatures. The proxy server supports two signature formats via the
`X-Signature-Format` header:

- `raw` (default): Standard ECDSA signature (for backend/CLI usage)
- `eip191`: EIP-191 prefixed signature (for browser wallet integration via `personal_sign`)

#### Query Request Format

Clients send JSON requests to the proxy server:

```json
{
  "bytes": "hex_encoded_query_request",
  "signature": "hex_encoded_signature"
}
```

The query request bytes should include:
- The query details (chain, contract, method, etc.)
- Optional `StakerAddress` field (20 bytes) for delegation scenarios

#### Rate Limiting Behavior

When staking-based rate limiting is enabled (either `--stakingPoolAddresses` or `--ccqFactoryAddress` is provided):

1. The proxy verifies the signature to recover the signer's address
2. If a `StakerAddress` is provided, the proxy checks if the signer is authorized to use that staker's limits
3. The proxy fetches the staking policy from the configured pools (with caching)
   - If `--stakingPoolAddresses` is used (preferred), pools are queried directly
   - If `--ccqFactoryAddress` is used, pools are discovered via the factory contract
4. Rate limits are enforced based on query types in the staking policy
5. Requests exceeding rate limits receive a `429 Too Many Requests` response

You must provide either `--stakingPoolAddresses` (preferred) or `--ccqFactoryAddress` to enable staking-based
rate limiting.

#### Supported Query Types

The proxy server supports all query types defined in the Wormhole Queries protocol:

**EVM Queries:**
- `ethCall` - Query contract state at current block
- `ethCallByTimestamp` - Query contract state at a specific timestamp
- `ethCallWithFinality` - Query contract state with finality guarantees

**Solana Queries:**
- `solAccount` - Query Solana account data
- `solPDA` - Query Solana Program Derived Address

For details on these query types, see the [Wormhole Queries Whitepaper](../whitepapers/0013_ccq.md).

## Telemetry

The proxy server provides two types of telemetry data, logs and metrics.

### Logging

The proxy server uses the same logging mechanism as the guardian. It will write to a local file, but can also be configured to
publish logs to Grafana using the Loki protocol. If you will be running your proxy server in mainnet, you should contact foundation
personnel about getting a Grafana ID to be used for logging and use it to set the `--telemetryLokiURL` command line argument.

If you set the log level to `info`, the proxy server logs information on all incoming requests and output bound responses. This can
be helpful for determining when requests reach quorum, but may be too chatty as the level of queries traffic grows. If that is the
case, you can set the log level to `warn`.

### Metrics

The proxy server uses Prometheus to track various activity and can publish them to Grafana. If you will be running your proxy server in mainnet,
you should contact foundation personnel about getting a Grafana ID and use it to set the `--promRemoteURL` command line argument.

For the set of available metrics, see [here](../node/cmd/ccq/metrics.go).

## Troubleshooting

### P2P Health

If you think you are having trouble with your access to the P2P network, you can add `--monitorPeers` to the command line arguments,
which will cause the proxy server to periodically check its connectivity to the P2P bootstrap peers, and attempt to reconnect if necessary.

### Invalid Requests

If the proxy server determines that a request is invalid, it does the following:

- Logs an error message with the signer address (and staker address if applicable).
- Increments the appropriate Prometheus metric.
- Sends a failure response to the client.

Note that if the proxy server thinks a request is valid, but the guardians do not, the guardians silently drop the request, so it will look
like a timeout. This is to avoid a denial of service attack on the guardians. This can happen if the proxy server is not properly permissioned
on the guardians.

### Request Logging

The proxy server logs all incoming requests with the signer address (and staker address if delegation is used).
At the `info` log level, you'll see logs like:

```
received request from client userId=signer:0x1234... requestId=abcd...
```

Or for delegated requests:

```
received request from client userId=delegated:0x5678...->staker:0x1234... requestId=abcd...
```
