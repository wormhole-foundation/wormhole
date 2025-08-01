# Usage

`make test` builds and runs the tests for Solana and EVM implementations.

# EVM

## Verify 13 signatures multisig (v1 VAA)

| Gas cost| Gas cost / VAA|        Implementation | Notes                                                       |
|--------:|--------------:|-----------------------|-------------------------------------------------------------|
| 134,689 |       134,689 |          mainnet core |                                                             |
| 108,341 |       108,341 |       [CoreBridgeLib] | Usage of this implementation is up to integrator.           |
|  88,686 |        88,686 | modified mainnet core | Keeps public keys in calldata. Not backwards compatible.[^1]|
|  69,570 |        69,570 | modified mainnet core | Optimized, backwards compatible implementation.[^1]         |
|  51,061 |        51,061 |        VerificationV2 | VAA with 100 bytes body                                     |
|  52,886 |        52,886 |        VerificationV2 | VAA with 5000 bytes body                                    |
|  50,836 |        50,836 |        VerificationV2 | Header + digest verification                                |
| 188,320 |        47,080 |        VerificationV2 | Header + digest batch verification for 4 VAAs               |

This means that there is a fixed overhead of ~5008 gas units for batch multisig verification in VerificationV2.

![Chart comparing gas costs for v1 VAA multisig verification across several implementations][v1 costs chart]


## Verify 1 threshold signature (v2 VAA)

| Gas cost| Gas cost / VAA|       Implementation |  Notes                                        |
|--------:|--------------:|----------------------|-----------------------------------------------|
| 13,874  |        13,874 | early VerificationV2 | Proxied                                       |
|  8,962  |         8,962 | early VerificationV2 | No proxy, i.e. unupgradeable                  |
|  8,544  |         8,544 |       VerificationV2 | VAA with 100 bytes body                       |
| 10,430  |        10,430 |       VerificationV2 | VAA with 5000 bytes body                      |
|  6,177  |         6,177 |       VerificationV2 | Header + digest verification                  |
| 17,385  |         4,347 |       VerificationV2 | Header + digest batch verification for 4 VAAs |

This means that there is a fixed overhead of ~2441 gas units for batch schnorr verification in VerificationV2.

![Chart comparing gas costs for v2 VAA threshold verification across several implementations][v2 costs chart]


## Current vs VerificationV2 comparison

![Chart comparing gas costs for current Wormhole Core and VerificationV2][EVM current vs VerificationV2 chart]

## Sources

The costs for implementations other than VerificationV2 come from [here](https://github.com/nonergodic/core-bridge/blob/fc4d76a/README.md)

The costs for VerificationV2 come from some benchmark tests that we have [here](test/TestAssembly2.sol#L433).

# Solana

VerificationV2 in Solana only implements verification of v2 VAAs.

## Performance

We did comparisons against verification implementations for v1 VAAs and an early version of VerificationV2.


| Computation cost| Unrecoverable Rent | # of Txs |       Implementation |  Notes                                                           |
|----------------:|-------------------:|---------:|----------------------|------------------------------------------------------------------|
|         146,709 |        0.003874272 |        4 |             old core | Only v1 VAAs.                                                    |
|         337,883 |        0.000015040 |        2 |          shim verify | Allows verification of large VAAs. Only v1 VAAs.                 |
|          53,417 |                  0 |        1 | early VerificationV2 | VAA with 100 bytes body.                                         |
|          33,902 |                  0 |        1 |       VerificationV2 | VAA with 100 bytes body.                                         |
|          34,057 |                  0 |        1 |       VerificationV2 | VAA with 100 bytes body. Also returns VAA body to caller.        |
|          33,378 |                  0 |        1 |       VerificationV2 | Header + digest verification. Allows verification of large VAAs. |

We created two charts here to show the verification costs of four implementations in two different ranges of priority prices.

The first chart shows the verification costs for priority prices of up to 12 lamports per computation unit.

The second chart shows the verification costs for priority prices of up to 30 millilamports per computation unit.

### Wide range of priority prices

![Chart comparing verification costs for current Wormhole Core, Shim Verify and VerificationV2][Solana current vs VerificationV2 chart]


### Lower range of priority prices

![Chart comparing verification costs for current Shim Verify and VerificationV2][Solana current vs VerificationV2, lower end chart]

## Sources

The costs for the old core and shim verify come from [here](https://github.com/wormhole-foundation/wormhole/tree/main/svm/wormhole-core-shims/programs/verify-vaa#performance-impact).
We used the total costs for both.

The costs for the Solana VerificationV2 implementation come from benchmark tests we have [here](src/solana/tests/verification_v2.ts#L432).

[^1]: Never deployed to mainnet nor testnet.

[CoreBridgeLib]: https://github.com/wormhole-foundation/wormhole-solidity-sdk/blob/main/src/libraries/CoreBridge.sol#L42

[v1 costs chart]:                      https://github.com/xLabs/core-bridge/blob/develop/data/performance-v1.svg?sanitize=true
[v2 costs chart]:                      https://github.com/xLabs/core-bridge/blob/develop/data/performance-v2.svg?sanitize=true
[EVM current vs VerificationV2 chart]: https://github.com/xLabs/core-bridge/blob/develop/data/performance-current-vs-verificationV2.svg?sanitize=true

[Solana current vs VerificationV2 chart]:            https://github.com/xLabs/core-bridge/blob/develop/data/solana-performance-current-vs-new.svg?sanitize=true&v=2
[Solana current vs VerificationV2, lower end chart]: https://github.com/xLabs/core-bridge/blob/develop/data/solana-performance-current-vs-new-lower-range.svg?sanitize=true&v=2
