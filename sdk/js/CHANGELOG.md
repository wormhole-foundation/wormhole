# Changelog

## 0.1.6

### Added

added parseSequencesFromLog\*

Terra NFT token bridge

getIsTransferCompleted on NFT bridge

export for wasm, createPostVaaInstructionSolana, createVerifySignaturesInstructionsSolana, postVaaSolana, postVaaSolanaWithRetry, and getSignedVAAWithRetry

re-export top level objects ethers_contracts, solana, terra, rpc, utils, bridge, token_bridge, nft_bridge

## 0.1.5

### Added

added postVaaSolanaWithRetry, which will retry transactions which failed during processing

added createVerifySignaturesInstructions, createPostVaaInstruction, which allows users to construct the postVaa process for themselves at the instruction level

added chunks and sendAndConfirmTransactionsWithRetry as utility functions

added integration tests for postVaaSolanaWithRetry

initial Oasis support

### Changed

deprecated postVaaSolana

## 0.1.4

initial AVAX testnet support

## 0.1.3

### Added

getSignedVAAHash

getIsTransferCompleted

## 0.1.1

### Added

CHAIN_ID_ETHEREUM_ROPSTEN

## 0.1.0

### Added

separate cjs and esm builds

updateWrappedOnSolana

top-level export getSignedVAAWithRetry

## 0.0.10

### Added

uint8ArrayToNative utility function for converting to native addresses from the uint8 format

Include node target wasms in lib

## 0.0.9

### Added

Integration tests

NodeJS target wasm

Ability to update attestations on EVM chains & Terra.

nativeToHexString utility function for converting native addresses into VAA hex format.

## 0.0.8

### Added

Polygon ChainId

## 0.0.7

### Changed

Changed function signature of attestFromTerra to be consistent with other terra functions

Removed hardcoded fees on terra transactions

## 0.0.6

### Changed

Allow separate payer and owner for Solana transfers

Support multiple EVM chains

Support native Terra tokens

Fixed nft_bridge getForeignAssetEth

## 0.0.5

### Added

NFT Bridge Support

getClaimAddressSolana

createMetaOnSolana

## 0.0.4

### Added

redeemOnEthNative

transferFromEthNative

## 0.0.3

### Added

Migration

NFT Bridge

### Changed

Fixed number overflow

Fixed guardian set index

## 0.0.2

Fix move postinstall to build

## 0.0.1

Initial release
