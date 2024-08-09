# Contract verification

The various EVM explorer sites (etherscan, bscscan, etc.) support contract
verification. This essentially entails uploading the source code to the site,
and they verify that the uploaded source code compiles to the same bytecode
that's actually deployed. This enables the explorer to properly parse the
transaction payloads according to the contract ABI.

This document outlines the process of verification. In general, you will need an
API key for the relevant explorer (this can be obtained by creating an account)
and to know at which address the contract code lives. The API key is expected to
be set in the `ETHERSCAN_API_KEY` environment variable for all APIs (not just
etherscan, bit of a misnomer).

Our contracts are structured as a separate proxy and an implementation. Both of
these components need to be verified, but the proxy contract only needs this
once, since it's not going to change. The implementation contract needs to be
verified each time it's upgraded.

## Verifying the proxy contract (first time)

The proxy contract is called `TokenBridge`. To verify it on e.g. Ethereum, at contract address `0x3ee18B2214AFF97000D974cf647E7C347E8fa585`, run

```bash
forge verify-contract --etherscan-api-key $ETHERSCAN_API_KEY --verifier-url "https://api.etherscan.io/api" 0x3ee18B2214AFF97000D974cf647E7C347E8fa585 contracts/bridge/TokenBridge.sol:TokenBridge --watch
```

## Verifying the implementation contract (on each upgrade)

To verify the actual implementation, at address `0x381752f5458282d317d12c30d2bd4d6e1fd8841e`, run

```bash
forge verify-contract --etherscan-api-key $ETHERSCAN_API_KEY --verifier-url "https://api.etherscan.io/api" 0x381752f5458282d317d12c30d2bd4d6e1fd8841e contracts/bridge/BridgeImplementation.sol:BridgeImplementation --watch
```

As a final step, when first registering the proxy contract, we need to verify
that it's a proxy that points to the implementation we just verified. This can
be done on ethereum at https://etherscan.io/proxyContractChecker
(other evm scanner sites have an identical page).
