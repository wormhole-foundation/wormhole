# Usage

`make test` builds and runs the tests for Solana and EVM implementations.

# EVM

## Verify 13 signatures multisig (v1 VAA)

| Gas cost| Gas cost / VAA|        Implementation | Notes                                                       |
|--------:|--------------:|-----------------------|-------------------------------------------------------------|
| 134,689 |       134,689 |          mainnet core |                                                             |
| 108,341 |       108,341 |         CoreBridgeLib | Usage of this implementation is up to integrator.           |
|  88,686 |        88,686 | modified mainnet core | Keeps public keys in calldata. Not backwards compatible.[1] |
|  69,570 |        69,570 | modified mainnet core | Optimized, backwards compatible implementation.[1]          |
|  51,061 |        51,061 |        VerificationV2 | VAA with 100 bytes body                                     |
|  52,886 |        52,886 |        VerificationV2 | VAA with 5000 bytes body                                    |
|  50,836 |        50,836 |        VerificationV2 | Header + digest verification                                |
| 188,320 |        47,080 |        VerificationV2 | Header + digest batch verification for 4 VAAs               |

This means that there is a fixed overhead of ~5008 gas units for batch multisig verification in VerificationV2.



## Verify 1 threshold signature (v2 VAA)

| Gas cost| Gas cost / VAA|        Implementation |  Notes                                        |
|--------:|--------------:|-----------------------|-----------------------------------------------|
| 13,874  |        13,874 | early VerificationV2  | Proxied                                       |
|  8,962  |         8,962 | early VerificationV2  | No proxy, i.e. unupgradeable                  |
|  8,544  |         8,544 |       VerificationV2  | VAA with 100 bytes body                       |
| 10,430  |        10,430 |       VerificationV2  | VAA with 5000 bytes body                      |
|  6,177  |         6,177 |       VerificationV2  | Header + digest verification                  |
| 17,385  |         4,347 |       VerificationV2  | Header + digest batch verification for 4 VAAs |

This means that there is a fixed overhead of ~2441 gas units for batch schnorr verification in VerificationV2.

## Sources

The costs for implementations other than VerificationV2 come from [here](https://github.com/nonergodic/core-bridge/blob/fc4d76a/README.md)

The costs for VerificationV2 come from some benchmark tests that we have [here](test/TestAssembly2.sol#L433).

[1] Never deployed to mainnet nor testnet.
