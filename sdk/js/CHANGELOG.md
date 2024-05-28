# Changelog

## 0.10.17

Add Mantle mainnet support

## 0.10.16

Add X Layer mainnet support

## 0.10.15

Add Provenance to cosmwasm chains
Add Scroll mainnet support
Add Blast mainnet support

## 0.10.14

Support for move of relayer ethereum contracts
Blast Sepolia support
XLayer Sepolia support
Linea Sepolia support
Berachain Testnet support

## 0.10.13

Mantle Sepolia support

## 0.10.12

Polygon Sepolia support

## 0.10.11

### Changes

Add Dymension to cosmwasm chains

base64 encode payload being sent to injective token bridge

### Removed

getSignedBatchVAA, getSignedBatchVAAWithRetry

## 0.10.10

### Changes

Added CCTP Sepolia chains support

## 0.10.9

### Changes

Algorand: prevent creation of unnecessary transactions

## 0.10.8

### Added

Holesky support

## 0.10.7

### Added

Arbitrum on Sepolia support

Base on Sepolia support

Optimism on Sepolia support

## 0.10.6

### Added

Celestia support

Scroll testnet support

### Changes

Solana WrappedMeta deserialization fix

## 0.10.5

### Changes

Auto relayer better status command

## 0.10.4

### Changes

AutoRelayer: v1.1 Release

Redeem on Algorand dynamic cost budget fix

Fixes id check in token helper for sui

## 0.10.3

### Added

Kujira chain support

## 0.10.2

### Added

Transfer from Aptos with payload support

### Changes

transferFromAptos payload type changed from string to Uint8Array

## 0.9.24

### Changes

Transfer from Sui with payload uses oldest EmitterCap _or_ creates a new one if none exist

## 0.9.23

### Changes

Bumped algosdk to 2.4.0

## 0.9.22

### Added

Base mainnet contract addresses

## 0.9.21

### Changes

Relayer status function improvements

Algorand changes for 3.16.2

## 0.9.20

### Added

Generic relayer support

### Changed

Updated terra.js version

## 0.9.18

### Added

Add support for Sei.

### Changed

injective parseSmartContractStateResponse fix

## 0.9.17

### Changed

Normalize Sui types

`unnormalizeSuiType` renamed to `trimSuiType`

## 0.9.16

### Changed

Sui redeem fix

## 0.9.15

### Added

Sui mainnet support

## 0.9.14

### Added

Sei testnet support

## 0.9.13

### Added

"sideEffects": false

### Changed

injective dependencies updated

## 0.9.12

### Added

Sepolia testnet support

## 0.9.11

### Added

Base testnet support

## 0.9.10

## Added

Aptos NFT bridge support

## 0.9.9

## Changed

Use BN.toArrayLike for compatibility with browserify and similar tools in `tokenIdToMint` function

## 0.9.8

## Changed

Use BN.toArrayLike for compatibility with browserify and similar tools

## 0.9.7

## Added

solana instruction decoder

## 0.9.6

## Changed

injective dependencies updated

solana token bridge cpi account fixes

solana account and instruction serialization fixes

## 0.9.5

### Added

injective mainnet addresses

## 0.9.4

## Changed

Neon testnet addresses

## 0.9.3

Fix `transferFromSolana`, `transferNativeSol` and `redeemOnSolana` for Token Bridge.

## 0.9.1

### Added

queryExternalIdInjective

parseSmartContractStateResponse

## 0.9.0

### Added

Methods to create transaction instructions for Wormhole (Core Bridge), Token Bridge and NFT Bridge

Methods to generate PDAs for Wormhole (Core Bridge), Token Bridge and NFT Bridge

Methods to deserialize account data for Wormhole (Core Bridge), Token Bridge and NFT Bridge

Other Solana utility objects and methods

VAA (Verified Wormhole Message) deserializers

Optional Confirmation arguments for account retrieval and wherever they are relevant

Mock objects to be used in local integration tests (e.g. MockGuardians)

### Changed

Use FQTs in Aptos SDK

### Removed

Dependency: @certusone/wormhole-sdk-wasm

Removed support for Ropsten since the chain has been deprecated.

## 0.8.0

### Added

Aptos support

### Changed

Wormchain rename

## 0.7.2

### Added

XPLA mainnet support and functions

## 0.7.1

### Added

Neon and XPLA testnet addresses

## 0.7.0

### Added

Near mainnet support

Injective testnet support

getSignedBatchVAA

getIsTransferCompletedTerra2

## 0.6.2

### Added

Algorand mainnet support

Updated consts.ts file
Exported signSendAndConfirmAlgorand()

## 0.6.1

### Added

getGovernorIsVAAEnqueued function

## 0.6.0

### Added

Wormhole chain devnet support

human-readable part parameter to `humanAddress` function

### Changed

`canonicalAddress` and `humanAddress` functions moved from terra to cosmos module

## 0.5.2

### Added

Support for PythNet
Chain ids for Arbitrum, Optimism, and Gnosis

## 0.5.1

### Added

Chain ids for Injective, Osmosis, Sui, and Aptos

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
