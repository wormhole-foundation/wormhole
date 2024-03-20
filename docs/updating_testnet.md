# Updating All Contracts in Testnet
This document describes how to update all the token bridge contracts in testnet for the purpose of resetting an emitter address
for a chain. In this example, we will be updating "all" token bridge contracts to reset the emitter address for
Sui. This is necessary because the Sui testnet environment gets reset periodically, requiring that the Wormhole
contracts be redeployed, changing the emitter address.

It should be noted that "all" does not literally mean all chains. At this time, this process does not support the following:
- Algorand
- Near

Supporting these chains just requires adding the appropriate documentation / scripting to support a contract upgrade
to reset a given emitter address.

Additionally, some chains allow you to overwrite an existing registration with a new one, so you do not need to upgrade the contract to reset a registration. This includes:
- Solana
- Aptos
- Sui

It should also be noted that this is a work in progress. Over time, some of these steps may be automated in the `worm` tool or other scripts.

## Process Setup
To set up for this process, start with a clean repo and follow these steps.
1. Checkout the `deploy_update_all_in_testnet` branch.
2. Rebase the branch off the latest `main`.

## Upgrading EVM Chains
The EVM contracts do not allow updating a registration for a chain that is already registered, so you need to upgrade the contract to clear the registration, and then submit the new VAA.
1. Edit `ethereum/contracts/bridge/BridgeImplementation.sol` to set the chain ID that you want reset (default is chain 21 for Sui).
2. cd to `ethereum` and do `npm run build`
3. Update the gas parameters in `ethereum/truffle-config.js` for Karura and Acala.
   - Use the `getKaruraTestnetGas.sh` and `getAcalaTestnetGas.sh` scripts to query for the latest gas prices and update the `gasPrice` and `gas` parameters for karura_testnet and acala_testnet. TODO: Automate this or put these scripts in the repo.
4. Run `./upgrade_all_test`. This should upgrade each EVM chain and submit the contract upgrade VAA for it.
5. Deal with any chains that fail, on a case by case basis. If you need to rerun a single chain, do something like this:
```bash
MNEMONIC=<deployerMnemonic> GUARDIAN_MNEMONIC=<guardianSecretKey> ./upgrade testnet TokenBridge acala
```

## Upgrading Cosmwasm Contracts
The cosmwasm contracts do not allow updating a registration for a chain that is already registered, so you need to upgrade the contract to clear the registration, and then submit the new VAA.
1. Edit `cosmwasm/contracts/token-bridge/src/contract.rs` to set the chain ID that you want reset (default is chain 21 for Sui).
2. cd to `cosmwasm` and do `make artifacts`

### Resetting the Registration on Terra2
1. Deploy the new code to Terra2 by doing the following. This should give you the `code_id`.
```bash
cd cosmwasm
node deployment/terra2/tools/deploy_single.js --network testnet --artifact artifacts/cw_token_bridge.wasm --mnemonic "<testnetDeployerSeedPhrase>"
```
2. The admin of the Terra2 token bridge in testnet has not been transferred to the contract itself, so you cannot use the standard contract upgrade VAA to complete the upgrade. Instead, do the following:
<!-- cspell:disable -->
```bash
node deployment/terra2/tools/migrate_testnet.js --network testnet --code_id <codeIdFromDeploy> --contract terra1c02vds4uhgtrmcw7ldlg75zumdqxr8hwf7npseuf2h58jzhpgjxsgmwkvk --mnemonic "<testnetDeployerSeedPhrase>"
```
<!-- cspell:enable -->

### Resetting the Registration on XPLA
1. Deploy the new code to XPLA by doing the following. This should give you the `code_id`.
```bash
cd cosmwasm
node deployment/xpla/tools/deploy_single.js --network testnet --artifact artifacts/cw_token_bridge.wasm --mnemonic "<testnetDeployerSeedPhrase>"
```
2. Generate the contract upgrade VAA.
```
export UPGRADE_VAA=`worm generate upgrade --guardian-secret <guardianSecretKey> --chain xpla --module TokenBridge --contract-address <codeIdFromDeploy>`
```
3. Submit the contract upgrade VAA.
```
worm submit $UPGRADE_VAA --network testnet --chain xpla
```

## Submitting the Chain Registration on each Chain
1. Generate the new chain registration VAA using the new emitter address (which for Sui is different from the contract address) and update the entry in the Testnet V2 notion page:
```bash
export VAA=`worm generate registration --guardian-secret <guardianSecretKey> --chain sui --module TokenBridge --contract-address <newEmitterAddress>`
```
2. Submit the VAA to each chain like this:
```bash
worm submit $VAA --network testnet --chain <chainName>
```

## Verifying the Registrations
To verify that the registrations are all correct for a given chain, you can do the following. This command currently works for Solana, EVM, Terra2, XPLA, Aptos, Sui and Sei.
```bash
worm info registrations testnet acala TokenBridge --verify
```
