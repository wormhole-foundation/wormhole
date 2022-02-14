# Pyth2wormhole relay example
IMPORTANT: This is not ready for production.

This package is an example Pyth2wormhole relayer implementation. The
main focus is to provide an automated integration test that will
perform last-mile delivery of Pyth2wormhole price attestations.

# How it works
## Relayer recap
When attesting with Wormhole, the final step consists of a query for
the guardian-signed attestation data on the guardian public RPC,
followed by posting the data to each desired target chain
contract. Each target chain contract lets callers verify the payload's
signatures, thus proving its validity. This activity means being
a Wormhole **relayer**.

## How this package relays attestations
`p2w-relay` is a Node.js relayer script targeting ETH that will
periodically query its source-chain counterpart for new sequence
numbers to query from the guardians. Any pending sequence numbers will
stick around in a global state until their corresponding messages are
successfully retrieved from the guardians. Later, target chain calls
are made and a given seqno is deleted from the pool. Failed target
chain calls will not be retried.
