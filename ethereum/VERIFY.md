# Contract verification

The various EVM explorer sites (etherscan, bscscan, etc.) support contract
verification. This essentially entails uploading the source code to the site,
and they verify that the uploaded source code compiles to the same bytecode
that's actually deployed. This enables the explorer to properly parse the
transaction payloads according to the contract ABI.

This document outlines the process of verification. In general, you will need an
API key for the relevant explorer (this can be obtained by creating an account)
and to know at which address the contract code lives. The API key is expected to
be set in the `ETHERSCAN_KEY` environment variable for all APIs (not just
etherscan, bit of a misnomer).

Our contracts are structured as a separate proxy and an implementation. Both of
these components need to be verified, but the proxy contract only needs this
once, since it's not going to change. The implementation contract needs to be
verified each time it's upgraded.

## Verifying the proxy contract (first time)

The proxy contract is called `TokenBridge`. To verify it on e.g. avalanche, at contract address `0x0e082F06FF657D94310cB8cE8B0D9a04541d8052`, run

```
ETHERSCAN_KEY=... npm run verify --module=TokenBridge --contract_address=0x0e082F06FF657D94310cB8cE8B0D9a04541d8052 --network=avalanche
```

(Note: the network name comes from the `truffle-config.json`).
(Note: In this case, the `ETHERSCAN_KEY` is your snowtrace API key).


## Verifying the implementation contract (on each upgrade)

To verify the actual implementation, at address `0xa321448d90d4e5b0a732867c18ea198e75cac48e`, run

```sh
ETHERSCAN_KEY=... npm run verify --module=BridgeImplementation --contract_address=0xa321448d90d4e5b0a732867c18ea198e75cac48e --network=avalanche
```

As a final step, when first registering the proxy contract, we need to verify
that it's a proxy that points to the implementation we just verified. This can
be done on avalanche at
https://snowtrace.io/proxyContractChecker?a=0x0e082F06FF657D94310cB8cE8B0D9a04541d8052

(other evm scanner sites have an identical page).


# Note
The `npm run verify` script uses the `truffle-plugin-verify` plugin under the
hood.  The version of `truffle-plugin-verify` pinned in the repo (`^0.5.11` at
the time of writing) doesn't support the avalanche RPC. In later versions of the
plugin, support was added, but other stuff has changed as well in the transitive
dependencies, so it fails to parse the `HDWallet` arguments in our
`truffle-config.json`. As a quick workaround, we backport the patch to `0.5.11`
by applying the `truffle-verify-constants.patch` file, which the `npm run
verify` script does transparently. Once the toolchain has been upgraded and the
errors fixed, this patch can be removed.
