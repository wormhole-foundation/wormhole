# Changelog

## 0.5.0

### Changed

Use `@certusone/wormhole-sdk-proto-web` and `@certusone/wormhole-sdk-wasm` packages

## 0.4.5

### Changed

Fix hex/Uint8Array to native Terra 2 for 20-byte addresses

## 0.4.4

### Added

Terra 2 mainnet addresses

## 0.4.3

### Added

Terra 2 testnet addresses

## 0.4.2

### Added

Neon testnet support
Terra 2 devnet support

### Changed

Updated terra.js

## 0.3.8

### Added

Neon testnet support

## 0.3.7

### Added

Acala mainnet support

## 0.3.6

### Changed

Fixed Algorand for addresses for non native assets

## 0.3.5

### Added

Added APIs to send transfers with payloads

## 0.3.4

### Changed

Fixed createWrappedAlgorand for Chain IDs > 128

## 0.3.3

### Added

Changed the payload3 support on Algorand conform to the new ABI specifications

Klaytn and Celo mainnet support

## 0.3.2

### Added

Payload 3 (Contract-Controlled Transfer) support

## 0.3.1

### Added

Moonbeam support

## 0.3.0

### Added

Added `tryNativeToHexString`

Added `tryNativeToUint8Array`

Added `tryHexToNativeString`

Added `tryUint8ArrayToNative`

Added support for passing in chain names wherever a chain is expected

Added chain id 0 (unset)

Added contract addresses to the `consts` module

### Changed

Deprecated `nativeToHexString`

Deprecated `hexToNativeString`

Deprecated `hexToNativeAssetString`

Deprecated `uint8ArrayToNative`

`isEVMChain` now performs type narrowing

`CHAIN_ID_*` constants now have literal types

## 0.2.7

### Added

safeBigIntToNumber() utility function

## 0.2.6

### Added

Algorand support

Celo support

## 0.2.5

### Changed

postVaa uses guardian_set_index from the vaa

## 0.2.4

### Added

Klaytn support

## 0.2.3

### Added

Expose feeRecipientAddress for redeemOnSolana

## 0.2.2

### Added

Include fee in parseTransferPayload

## 0.2.1

### Added

Default relayerFee parameter (defaults to 0) to each token bridge transfer function

Expose overrides parameter for signer \*Eth functions

Karura support

Acala support

## 0.2.0

### Changed

Updated @terra-money/terra.js to 3.0.7

Removed @terra-money/wallet-provider

Removed walletAddress parameter from getIsTransferCompletedTerra

## 0.1.7

### Added

Fantom support

Aurora support

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
