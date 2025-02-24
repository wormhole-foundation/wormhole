# Transfer Verifier - CLI tool

This package can be used to run the Transfer Verifier as a standalone tool. This allows for quick iteration: a developer can
modify the package code at `node/pkg/txverifier/` and run this tool to test the changes against either mainnet or a local network.

## Usage

### Ethereum

Ensure that you have a valid API key connected with a **WebSockets** URL. 

The following command runs the Transfer Verifier against mainnet.

```sh
# Run from the root of the Wormhole monorepo. This script uses the mainnet values for the core contracts.
./build/bin/guardiand transfer-verifier evm \
    --rpcUrl $RPC_URL \
    --coreContract 0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B \
    --tokenContract 0x3ee18B2214AFF97000D974cf647E7C347E8fa585 \
    --wrappedNativeContract 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2 \
    --logLevel debug
```

To test against a forked local network, change the RPC URL to anvil's default (also used by the Tilt network), and update
the contract addresses.
