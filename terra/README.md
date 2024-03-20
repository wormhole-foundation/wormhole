# Terra Wormhole Contract Deployment

This readme describes the steps for building, verifying, and deploying Terra smart contracts for Wormhole.

**WARNING**: *This process is only Linux host compatible at this time.*

## Verify Tilt

Before building Terra contracts, ensure that the specific commit you will be
building from passes in tilt. This ensures the basic functionality of the
Terra smart contracts that you are about to build and deploy.

## Build Contracts

The following command can be used to build Terra contracts via Docker.

Build Target Options: [`mainnet`|`testnet`|`devnet`]

These network names correspond to the naming convention used by wormhole
elsewhere. This means that `mainnet` corresponds to Terra `mainnet`,
`testnet` corresponds to Terra `testnet`, and `devnet` is `localterra`.

```console
wormhole/terra $ make artifacts
```

Upon completion, the compiled bytecode for the Terra contracts will be placed
into the `artifacts` directory.

## Verify Checksums

Now that you have built the Terra contracts, you should ask a peer to build
using the same process and compare the equivalent checksums.txt files to make
sure the contract bytecode(s) are deterministic.

```console
wormhole/terra $ cat artifacts/checksums.txt
```

Once you have verified the Terra contracts are deterministic with a peer, you can now move to the deploy step.

## Run tests

**Disclaimer: Currently the only test that exists is for the token bridge's transfer.**

You can run the integration test suite on the artifacts you built.

```console
wormhole/terra $ make test
```

This command deploys your artifacts and performs various interactions with your
contracts in a LocalTerra node. Any new functionality (including expected errors)
to the contracts should be added to this test suite.

## Deploy Contracts

Now that you have built and verified checksums, you can now deploy one or more relevant contracts to the Terra blockchain.

Deploy Target Options: [`mainnet`|`testnet`|`devnet`]

You will need to define a `payer-DEPLOY_TARGET.json` for the relevant deploy
target (eg. `payer-testnet.json`).  This will contain the relevant wallet
private key that you will be using to deploy the contracts.

```console
wormhole/terra $ make deploy/bridge
wormhole/terra $ make deploy/token_bridge
```

For each deployed contract, you will get a code id for that relevant
contract for the deployment. The code id will be written to a network specific file
that you can read to execute further steps like contract verification.
The deployment prints the name of this file.

### Instantiation

The deployment script currently does not instantiate a new live contract but rather only uploads the bytecode.
This is all that is needed to upgrade existing contracts.

To bootstrap a new instance of the contract you may need to modify the deployment slightly.

## Verify On-Chain

Now that you have deployed one or more contracts on-chain, you can verify the
onchain bytecode and make sure it matches the same checksums you identified
above.

For each contract you wish to verify on-chain, you will need the following elements:

- Path to the contracts bytecode (eg. `artifacts-testnet/token_bridge.wasm`)
- Terra code id for the relevant contract (eg. `59614`)
- A network to verify on (`mainnet`, `testnet`, or `devnet`)

Below is how to verify all two contracts:

```console
wormhole/terra $ ./verify artifacts/wormhole.wasm NEW_BRIDGE_CODE_ID
wormhole/terra $ ./verify artifacts/token_bridge.wasm NEW_TOKEN_BRIDGE_CODE_ID
```
Example: `./verify artifacts/token_bridge.wasm 59614`

For each contract, you should expect a `Successfully verified` output message.
If all contracts can be successfully verified, you can engage in Wormhole
protocol governance to obtain an authorized VAA for the contract upgrade(s).

A verification failure should never happen, and is a sign of some error in the
deployment process.  Do not proceed with governance until you can verify the
on-chain bytecode with the locally compiled bytecode.


## Governance

### Mainnet

Upgrades on mainnet have to go through governance. Once the code is deployed in
the previous step, an unsigned governance VAA can be generated from the root of the repository:

```sh
token_bridge_id=$(cat token_bridge-code-id-mainnet.txt)
./scripts/contract-upgrade-governance.sh --module token_bridge --chain terra --address $token_bridge_id > "token-bridge-upgrade-${token_bridge_id}.prototxt"
```

Supposing that the token bridge code id is 59614,
this will write to the `token-bridge-upgrade-59614.prototxt` file, which can
now be shared with the guardians to vote on.

Once the guardians have reached quorum, the VAA may be submitted from any
funded wallet with `worm` CLI:

``` sh
export TERRA_MNEMONIC="..."
# Note that the chain should be inferred from the governance VAA
worm submit --network mainnet <signed VAA (hex without 0x prefix)> --rpc "https://terra-classic-lcd.publicnode.com"
```

### Testnet

For the contracts on testnet, the deployer wallet retains the upgrade
authority, so these don't have to go through governance.

For example, to migrate the token bridge to 59614, run in `tools/`:

<!-- cspell:disable -->
``` sh
node migrate_testnet.js --code_id 59614 --contract terra1pseddrv0yfsn76u4zxrjmtf45kdlmalswdv39a --mnemonic "..."
```
<!-- cspell:enable -->
