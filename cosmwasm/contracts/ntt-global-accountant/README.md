# Native Token Transfer - Global Accountant

This contract is intended to serve the same purpose as the [Global Accountant](../../../whitepapers/0011_accountant.md), but for Native Token Transfer protocols instead of the [Token Bridge](../../../whitepapers/0003_token_bridge.md).

This is made possible by the following modifications:

- Keep a list of [Standard Relayer](https://docs.wormhole.com/wormhole/explore-wormhole/relayer#standard-relayers) emitters.
  - These registrations are approved by Wormhole Governance akin to the Token Bridge, so this is a 1-1 replacement.
- Extract the message sender and payload from either a [Core Bridge message](../../../whitepapers/0001_generic_message_passing.md) or a Standard Relayer payload.
  - NTTs can either emit messages via their transceiver directly or via Standard Relayers.
  - This should be done in the accountant contract (as opposed to pre-processed by the guardian) in order to support backfilling signed VAAs.
- Keep a map of registered NTT transceivers and their locking hub.
  - This contract must be able to permissionlessly determine what constitutes a valid transfer between two NTT transceivers. In order to determine that, the contract needs to have a record of both the sending and receiving transceivers' registrations.
  - An NTT transceiver emits a VAA when...
    - it is initialized, which includes if it's associated manager is in `locking` or `burning` mode.
    - a new transceiver is registered with it.
  - These VAAs can then be relayed to the NTT Global Accountant, verified, parsed, and stored into a map.
  - These maps are a one-way lookup of...
    - Transceiver hubs `[chainA, emitter address] -> [chainB, foreign transceiver emitter address]`
    - Transceiver peers `[chainA, emitter address, chainB] -> [foreign transceiver emitter address]`
- Update the logic for handling an observation or NTT transfer VAA. Instead of checking the token bridge emitter:
  - If the core message was from a known Standard Relayer emitter, use the sender as the emitter, otherwise use the core message emitter.
  - If the emitter (sender) does not have a key in the maps, return.
  - If the emitter's foreign transceiver (known receiver) does not match the target recipient, return.
  - If the foreign transceiver (receiver) does not have a registration for the emitter (sender) - i.e. they are not cross-registered, return.
- Use `<chain, locking_hub_chain, locking_hub_transceiver_address>` in place of `<chain, token_chain, token_address>` for the `Account::key` to track each network of transceivers separately. This requires a 1:1 transceiver:token mapping.

The guardians will have a new allow list of NTTs and will be expected to submit observations for the allow-listed NTTs to the NTT global accountant contract on Wormhole Gateway.

## Caveats

1. The NTT Global Accountant can only account for NTTs which have _exactly_ one Manager in `lock` mode. In an all `burn` mode network, the tokens have a potentially unlimited supply on every chain or other legitimate methods of minting which cannot be known to the accountant. In a multiple `lock` setup, the existing accountant logic, which is single-source chain per "token", would no longer directly apply. It is possible for these cases to be covered in the future, but may require further NTT protocol and significant accountant logic changes.
1. The NTT Global Accountant expects each transceiver to be associated with exactly 1 token.
1. In order to avoid backfilling (the process of relaying VAAs emitted prior to the enforcement of Global Accountant on a network of NTTs), these initial steps should be completed, in order, before making any transfers.
   1. Initialize the NTT transceivers.
   1. Add the NTT transceivers' emitters to the Guardian's allow list.
   1. Cross-register the NTT transceivers on-chain, emitting the VAAs.
   1. Submit the locking mode initialization VAA to the NTT accountant contract.
   1. Submit the burn transceivers' locking hub registration VAA to the NTT accountant contract.
   1. Submit the remaining registration VAAs to the NTT accountant.
