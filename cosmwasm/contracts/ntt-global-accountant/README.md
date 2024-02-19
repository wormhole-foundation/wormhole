# Native Token Transfer - Global Accountant

This contract is intended to serve the same purpose as the [Global Accountant](../../../whitepapers/0011_accountant.md), but for Native Token Transfer protocols instead of the [Token Bridge](../../../whitepapers/0003_token_bridge.md).

This is made possible by the following modifications:

- Keep a list of [Standard Relayer](https://docs.wormhole.com/wormhole/explore-wormhole/relayer#standard-relayers) emitters.
  - These registrations are approved by Wormhole Governance akin to the Token Bridge, so this is a 1-1 replacement.
- Extract the message sender and payload from either a [Core Bridge message](../../../whitepapers/0001_generic_message_passing.md) or a Standard Relayer payload.
  - NTTs can either emit messages via their endpoint directly or via Standard Relayers.
  - This should be done in the accountant contract (as opposed to pre-processed by the guardian) in order to support backfilling signed VAAs.
- Keep a map of registered NTT endpoints.
  - This contract must be able to permissionlessly determine what constitutes a valid transfer between two NTT endpoints. In order to determine that, the contract needs to have a record of both the sending and receiving endpoints' registrations.
  - Ideally, the NTT endpoint emits a VAA when a new endpoint is registered with it, which can then be relayed to the NTT Global Accountant, verified, parsed, and stored into a map.
  - This map is a one-way lookup of `[chainA, emitter address] -> [chainB, foreign endpoint emitter address]`
- Update the logic for handling an observation or NTT transfer VAA. Instead of checking the token bridge emitter:
  - If the core message was from a known Standard Relayer emitter, use the sender as the emitter, otherwise use the core message emitter.
  - If the emitter (sender) does not have a key in the map, return.
  - If the emitter's foreign endpoint (known receiver) does not match the target recipient, return.
  - If the foreign endpoint (receiver) does not have a registration for the emitter (sender) - i.e. they are not cross-registered, return.
- Use `<chain, lock_chain, lock_endpoint_address, endpoint_address>` in place of `<chain, token_chain, token_address>` for the `Account::key` to track each endpoint separately. This requires a 1:1 endpoint:token mapping.

The guardians will have a new allow list of NTTs and will be expected to relay governance VAAs for the allow-listed NTTs to the NTT global accountant contract on Wormhole Gateway.

## Outstanding Questions

- How can the contract determine when to add or subtract from the token balance?
  - It will need some indication of which of the managers in the network is in `lock` mode.

## Caveats

1. The NTT Global Accountant can only account for NTTs which are in `lock` mode. In `burn` mode, the tokens have a potentially unlimited supply on every chain or other legitimate methods of minting which cannot be known to the accountant.
1. The NTT Global Accountant expects each endpoint to be associated with exactly 1 token.
1. In order to avoid backfilling (the process of relaying VAAs emitted prior to the enforcement of Global Accountant on a network of NTTs), these initial steps should be completed, in order, before making any transfers.
   1. Add the NTT endpoints' emitters to the Guardian's allow list.
   1. Cross-register the NTT endpoints on-chain, emitting the VAAs. (The guardians should relay these to the NTT Global Accountant)
