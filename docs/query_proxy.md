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

There are two main parts to configuring the proxy server. The first is setting up the command line arguments,
which generally will not change after initial setup. The second part of the configuration is the permissions file,
which will change as the requirements of integrators change.

### Proxy Server Command Line Arguments

The following is a sample command line for running the proxy server in mainnet.

```shell
wormhole $build/bin/guardiand query-server \
      --env "mainnet" \
      --nodeKey /home/ccq/data/ccq_server.nodeKey \
      --permFile "/home/ccq/data/ccq_server.perms.json" \
      --signerKey "/home/ccq/data/ccq_server.signerKey" \
      --listenAddr "[::]:8080" \
      --ethRPC https://eth.drpc.org \
      --ethContract "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B" \
      --logLevel=info \
      --telemetryLokiURL $LOKI_URL \
      --telemetryNodeName "Mainnet CCQ server 1" \
      --promRemoteURL $PROM_URL
```

- The `env` can be mainnet, testnet or devnet.
- The `nodeKey` should point to the file containing the P2P key. The first time the proxy runs, if the
  file does not exist, it will be created. You can look in the proxy server logs to get the generated key.
- The `permFile` is the JSON permissions file, which is documented below.
- The `signerKey` should point to an armored file containing a key that will be used to sign requests received
  from integrators who are configured to support auto signing and opt not to sign a request. Please see below
  for how to generate this file.
- The `listenAddr` specifies the port on which the proxy listens for REST requests.
- The `ethRPC` and `ethContract` are used to read the wormhole guardian set on start up. The address
  above is for mainnet. If you are running in testnet, you should point to Holesky and use `0xa10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a`.
  (You can confirm these addresses [here](https://docs.wormhole.com/wormhole/reference/constants#contract-addresses).)
  Note that using a public endpoint should be fine, since the proxy only does a single read of the guardian set.
- The `telemetryLokiURL`, `telemetryNodeName` and `promRemoteURL` are used for telemetry purposes and
  the values will be provided by Wormhole Foundation personnel if appropriate.

Optional Parameters

- The `gossipAdvertiseAddress` argument allows you to specify an external IP to advertize on P2P (use if behind a NAT or running in k8s).
- The `monitorPeers` flag will cause the proxy server to periodically check its connectivity to the P2P bootstrap peers, and attempt to reconnect if necessary.
- The `allowAnything` flag enables defining users with the `allowAnything` flag set to true. This is only allowed in testnet and devnet.

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

### Permissions Configuration

The file specified by the `permFile` parameter contains JSON that defines the set of allowed queries users, along with the
sets of requests they are allowed to make.

#### File Format

The simplest file would look something like this

```json
{
  "permissions": [
    {
      "userName": "Monitor",
      "apiKey": "insert_generated_api_key_here",
      "allowUnsigned": true,
      "allowedCalls": [
        {
          "ethCall": {
            "note:": "Name of WETH on Ethereum",
            "chain": 2,
            "contractAddress": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
            "call": "0x06fdde03"
          }
        }
      ]
    }
  ]
}
```

This creates a single user called "Monitor", who will use the specified API key (more on API keys below).
This user is allowed to submit unsigned requests (which will be signed using the configured signing key).

This sample user is only allowed to make a single `ethCall` request on Ethereum (Wormhole chain ID 2),
which allows them to call the `name` method on the contract that resides at `0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2`.
The `call` parameter is the first four bytes of the hash of the ABI encoded function call to be allowed.

A given user can have any number of allowed calls (at least one), but they can only make calls that are configured here.

#### Supported Call Types

The proxy server supports all of the query types supported by the Wormhole Queries protocol. For details on those calls,
please see the [Wormhole Queries Whitepaper](../whitepapers/0013_ccq.md).

The following are the EVM call types, all of which require the `chain`, `contractAddress` and `call` arguments.

- `ethCall`
- `ethCallByTimestamp`
- `ethCallWithFinality`

The following are the Solana call types. Both require the `chain` parameter plus the extra parameter listed below.

- `solAccount`, requires the `account` parameter.
- `solPDA`, requires the `programAddress` parameter.

The Solana account and and program address can be expressed as either a 32 byte hex string starting with "0x" or as a base 58 value.

#### Creating New API Keys

Each user must have an API key. These keys only have meaning to the proxy server. They are not passed to the guardians.
The proxy requires that a key be present in each query request, and that the specified key exists in the permissions file.
Beyond that, the API keys have no special meaning. They can be generated using a site like [this](https://www.uuidgenerator.net/version4).

#### Updating the Permissions File

The proxy server monitors the permissions file for changes. Whenever a change is detected, it reads the file, validates it, and if
it passes validation, switches to the new version. Care should be taken when editing the file while the proxy server is running, because
as soon as you save the file, the changes will be picked up (whether they are logically complete or not).

#### The `allowAnything` flag

If this flag is specified for a user, then that user may make any call on any supported chain, without restriction.
This flag is only allowed if the `allowAnything` command line argument is specified.
If this flag is specified, then `allowedCalls` must not be specified.

```json
{
  "permissions": [
    {
      "userName": "Monitor",
      "apiKey": "insert_generated_api_key_here",
      "allowUnsigned": true,
      "allowedAnything": true
    }
  ]
}
```

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

- Logs an error message using the user name (not the API Key).
- Increments the appropriate Prometheus metric.
- Sends a failure response to the user.

Note that if the proxy server thinks a request is valid, but the guardians do not, the guardians silently drop the request, so it will look
like a timeout. This is to avoid a denial of service attack on the guardians. This can happen if the proxy server is not properly permissioned
on the guardians.

### Logging Request Detail.

If a given integrator is reporting problems with their queries, you may find it useful to add the following to their permissions config
(at the same level as the API Key, etc).

```json
"logResponses": true,
```

This will cause the proxy server to log every response received for that user, along with the number of responses and how many are
still needed to meet quorum.
