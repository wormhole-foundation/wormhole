# Alternate Publisher

## Introduction

The Alternate Publisher provides a mechanism to publish observations (and potentially signed VAAs) to one or more HTTP endpoints
as a backup to the P2P gossip network. The guardian can be configured to publish to one or more endpoints, or none (disabling the feature).
Note that this is in addition to publishing to gossip. This feature does not impact gossip traffic.

For each endpoint, a list of chains may be specified, meaning only events for those chains are published to that endpoint (default is all chains).

Additionally, for each endpoint, a publish delay may be specified. If a delay is specified (the default is immediate), then observations
will be batched for that long before publishing. This should allow for reduced HTTP traffic.

A couple of possible use cases for this feature might be Pyth and Wormholescan.

- Pyth would probably want to only receive observations from PythNet and without any delay.
- Wormholescan would probably want to receive observations for all chains and wouldn't mind some delay to allow for batching.

## Configuration

The Alternate Publisher can be enabled by specifying one or more `--additionalPublishEndpoint` parameters in the guardiand config. Each instance of the parameter represents a single publishing endpoint and is defined as follows:

<!-- cspell:disable -->

```bash
--additionalPublishEndpoint label;url;delay;chains
```

<!-- cspell:enable -->

The fields are defined as follows:

- **label** is a string that is used to tag this endpoint in log messages and Prometheus metrics.
- **url** is the http server endpoint to which the guardian should connect and publish.
- **delay**, if specified (or non-zero) is the time the guardian should delay in order to batch observations. Zero / not set means publish immediately.
- **chains**, if specified, is a comma-separated list of emitter chain IDs or names for which observations should be forwarded. If not set, all chains will be published.

The **label** and **url** fields are required, but the **delay** and **chains** are optional.

If **chains** is specified, **delay** is required but you may leave it blank or specify "0" to publish immediately. It is valid to specify the **delay** without the **chains**,
meaning there are only three fields (without a final semicolon).

The **delay** is specified as a `time.Duration`. Please see [here](https://pkg.go.dev/time#ParseDuration) for a description of how to specify it.

Note that if the **chains** parameter begins with a dash (minus sign), then it means "publish everything **except** these chains.

### Configuration Example

For the Pyth and Wormholescan examples, the configuration might look like this.

<!-- cspell:disable -->

```bash
--additionalPublishEndpoint "pyth;http:pyth_endpoint_url;0;pythnet"
--additionalPublishEndpoint "wormholescan;http:wormholescan_endpoint;1s"
```

<!-- cspell:enable -->

This means we will immediately publish events for ChainIDPythNet to the Pyth endpoint, and we will publish all events to the Wormholescan
endpoint with a one second delay to allow for batching.

## Implementation

The Alternate Publisher uses an `http.Client` and HTTP POST to publish requests. It creates a pool of workers to allow for multiple parallel requests.
This means that observations may be received at an endpoint in a different order than they are published by a guardian. This should be acceptable.

If an endpoint is configured with a delay, it creates a worker routine to manage the batching before publishing requests to the worker pool.

For immediate publishing, it formats the request and writes it to the worker pool channel immediately.

### Object Layout

If alternate publishing is enabled, there will be a single `AlternatePublisher` object. It contains a list of `Endpoint` objects where each one represents a configured / enabled endpoint. A pointer to the `AlternatePublisher` (or nil) is passed into the processor, which will call into it to publish observations.

The `AlternatePublisher` object has a single `http.Client` and a pool of `httpWorker` routines fed by a single `httpWorkerChan` channel. The payload of that channel is an HTTP request. The workers pick requests off of the channel and post them in a blocking manner. They update metrics based on the result.

The `AlternatePublisher.PublishObservation` function loops through the endpoints. For each endpoint, it calls `shouldPublish` to see if the observation should be published based on the emitter chain ID. If the observation should be published, and the endpoint is not configured for batching, the HTTP request is formatted and posted to the `httpWorkerChan` for immediate publishing. Otherwise, the observation is posted to the `obsvBatchChan` channel on the endpoint for batching.

The `Endpoint` object contains the URL of the endpoint, the delay value, and a map of enabled chains. If the map is empty, then observations for all chains are published. Otherwise, the emitter chain ID must be in the map for the observation to be published. If the delay is zero, then observations are published immediately.

If the endpoint delay is non-zero, the `Endpoint` object has a `batchWorker` with a `obsvBatchChan` channel used to post to it. The `batchWorker` delays publishing to the `httpWorkerChan` to allow for batching. It uses the existing `common.ReadFromChannelWithTimeout` function to perform batching.

## Endpoint Interface

Messages are published using HTTP POST operations where the body is a protobuf encoded message. The posts have the `"Content-Type", "application/octet-stream"` header.

Currently the only messages being published on this interface are signed observations. They are published to `/SignedObservationBatch` where the body is a `gossipv1.SignedObservationBatch` message. This message can contain _up to_ `MaxObservationBatchSize` (4000) observations, although will most likely be much less than that.

## Testing

### Unit Tests

There are a variety of unit tests in `alternate_pub_test.go` which test individual functions.

### End to End Test

In addition to the unit tests, the test in `end2end_test.go` does a full end-to-end test of the pyth and wormholescan scenarios. It instantiates an `AlternatePublisher`
with two endpoints simulated by local HTTP server objects. It then publishes a bunch of observations and verifies that the endpoints received the correct results.

## Possible Future Enhancement

It may be desirable to also publish signed VAAs. To do that, we would add an `AlternatePublisher.PostSignedVAA` function and publish to `/SignedVAAWithQuorum`.
The payload could be a protobuf encoded `gossipv1.SignedVAAWithQuorum`.

We would need to decide if it makes sense to batch signed VAAs. If not, we could just format the HTTP request and post it to the `httpWorkerChan` like the immediate
observation case. If we do want to allow batching of signed VAAs, we could add a parallel channel and batch worker for it.

If we add support for signed VAAs, we may want to update the config parameter to allow publishing only some event types (observations vs. VAAs vs. both).

## TODO

Please see the prologue of [alternate_pub.go](alternate_pub.go) for the current list of outstanding issues.
