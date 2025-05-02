# Transfer Verifier - CLI tool

This package can be used to run the Transfer Verifier as a standalone tool. This allows for quick iteration: a developer can
modify the package code at `node/pkg/txverifier/` and run this tool to test the changes against either mainnet or a local network.

## Usage

### Ethereum

_Ensure that you have a valid API key connected with a **WebSockets** URL._

#### Devnet

The script at `scripts/transfer-verifier-localnet.sh` runs the transfer verifier against devnet.

#### Automated testing

This command provides a "sanity check" mode that runs the package against mainnet data using a hard-coded
list of tx hashes and their expected return values. It checks that ruling of true/false matches what's
expected, and ensures that the expected error code match.

The code will log a fatal error and exit as soon as one of the sanity checks fail.

This functions as a quick way to do regression testing using real data, avoiding the need for extensive mocking
of the RPC calls.

# Run from the root of the Wormhole monorepo. This script uses the mainnet values for the core contracts.
```sh
./build/bin/guardiand transfer-verifier evm \
    --rpcUrl $RPC_URL \
    --coreContract 0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B \
    --tokenContract 0x3ee18B2214AFF97000D974cf647E7C347E8fa585 \
    --wrappedNativeContract 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2 \
    --sanity=true
```


#### Manual testing

The following command runs the Transfer Verifier against mainnet. It will run until killed.

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

##### Testing a single receipt

A single receipt can be evaluated by adding the `--hash` flag and passing an Ethereum receipt hash.
This is the easiest way to get insight into how the algorithm works and to verify expected results.

Any unexpected results should be added to the sanity checks described earlier to help guard against regressions.

**Example hashes**:

Message publication with wrapped asset
- `0xa3e0bdf8896a0e1f1552eaa346a914d655a4f94a94739c4ffe86a941a47ec7a8`

Message publication with a deposit
- `0x173a027bb960fa2e2e2275c66649264c1b961ffae0fbb4082efdf329a701979a`

Many transfers, one event with no topics, and a LogMessagePublished event. 
Unrelated to the Token Bridge. Should be successfully parsed and ultimately skipped.
- `0x27acebf817c3c244adb47cd3867620d9a30691c0587c4f484878fa896068b4d5`

Mayan Swift transfer. Should be successfully parsed and ultimately skipped.
- `0xdfa07c6910e3650faa999986c4e85a0160eb7039f3697e4143a4a737e4036edd`

- `0xb6a993373786c962c864d57c77944b2c58056250e09fc6a15c87d473e5cfe206`

