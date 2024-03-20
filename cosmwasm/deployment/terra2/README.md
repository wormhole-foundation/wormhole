# Terra2 Wormhole Contract Deployment

This readme describes the steps for building, verifying, and deploying Terra2 smart contracts for Wormhole.

**WARNING**: _This process is only Linux host compatible at this time._

## Verify Tilt

Before building Terra contracts, ensure that the specific commit you will be
building from passes in tilt. This ensures the basic functionality of the
Terra smart contracts that you are about to build and deploy.

## Build Contracts

The following command can be used to build Terra2 contracts via Docker.

Build Target Options: [`mainnet`|`testnet`|`devnet`]

These network names correspond to the naming convention used by wormhole
elsewhere. This means that `mainnet` corresponds to Terra `mainnet`,
`testnet` corresponds to Terra `testnet`, and `devnet` is `localterra`.

```console
make artifacts
```

Upon completion, the compiled bytecode for the Terra contracts will be placed
into the `artifacts` directory.

## Verify Checksums

Now that you have built the Terra contracts, you should ask a peer to build
using the same process and compare the equivalent checksums.txt files to make
sure the contract bytecode(s) are deterministic.

```console
cat artifacts/checksums.txt
```

Once you have verified the Terra contracts are deterministic with a peer, you can now move to the deploy step.

## Run tests

**Disclaimer: Currently the only test that exists is for the token bridge's transfer.**

You can run the integration test suite on the artifacts you built.

```console
make test
```

This command deploys your artifacts and performs various interactions with your
contracts in a LocalTerra node. Any new functionality (including expected errors)
to the contracts should be added to this test suite.

## Deploy Contracts

Now that you have built and verified checksums, you can now deploy one or more relevant contracts to the Terra blockchain.

Deploy Target Options: [`mainnet`|`testnet`|`devnet`]

You will need to define a `payer-DEPLOY_TARGET.json` for the relevant deploy
target (eg. `payer-testnet.json`). This will contain the relevant wallet
private key that you will be using to deploy the contracts.

```console
make deploy/bridge
make deploy/token_bridge
make deploy/nft_bridge
```

For each deployed contract, you will get a code id for that relevant
contract for the deployment, make note of these so you can use them in
the next step for on-chain verification.

## Verify On-Chain

Now that you have deployed one or more contracts on-chain, you can verify the
onchain bytecode and make sure it matches the same checksums you identified
above.

For each contract you wish to verify on-chain, you will need the following elements:

- Path to the contracts bytecode (eg. `artifacts-testnet/token_bridge.wasm`)
- Terra code id for the relevant contract (eg. `59614`)
- A network to verify on (`mainnet`, `testnet`, or `devnet`)

Below is how to verify all three contracts:

```console
./verify artifacts/cw_wormhole.wasm NEW_BRIDGE_CODE_ID
./verify artifacts/cw_token_bridge.wasm NEW_TOKEN_BRIDGE_CODE_ID
./verify artifacts/nft_bridge.wasm NEW_NFT_BRIDGE_CODE_ID
```

Example: `./verify artifacts/token_bridge.wasm 59614`

For each contract, you should expect a `Successfully verified` output message.
If all contracts can be successfully verified, you can engage in Wormhole
protocol governance to obtain an authorized VAA for the contract upgrade(s).

A verification failure should never happen, and is a sign of some error in the
deployment process. Do not proceed with governance until you can verify the
on-chain bytecode with the locally compiled bytecode.

## Governance

### Mainnet

Upgrades on mainnet have to go through governance. Once the code is deployed in
the previous step, an unsigned governance VAA can be generated

```sh
./generate_governance -m token_bridge -c 59614 > token-bridge-upgrade-59614.prototxt
```

This will write to the `token-bridge-upgrade-59614.prototxt` file, which can
now be shared with the guardians to vote on.

Once the guardians have reached quorum, the VAA may be submitted from any
funded wallet: TODO - make this easier and more unified

<!-- cspell:disable -->
```sh
node main.js terra execute_governance_vaa <signed VAA (hex)> --rpc "https://lcd.terra.dev" --chain_id "columbus-5" --mnemonic "..." --token_bridge "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf"
```
<!-- cspell:enable -->

### Testnet

For the contracts on testnet, the deployer wallet retains the upgrade
authority, so these don't have to go through governance.

For example, to migrate the token bridge to 59614, run in `tools/`:

<!-- cspell:disable -->
```sh
node migrate_testnet.js --code_id 59614 --contract terra1pseddrv0yfsn76u4zxrjmtf45kdlmalswdv39a --mnemonic "..."
```
<!-- cspell:enable -->
