# Sui Wormhole Token Bridge Design

TODO: make sure this is up to date

The Token Bridge is responsible for storing treasury caps and locked tokens and exposing functions for initiating and completing transfers, which are gated behind VAAs. It also supports token attestations from foreign chains (which must be done once prior to transfer), contract upgrades, and chain registration.

## Token Attestation

TODO: up to date implementation notes

The sui RPC provides a way to get the object id for CoinMetadata objects:
https://github.com/MystenLabs/sui/pull/6281/files#diff-80bf625d87d89549275351d95cfdfab4a6c2a1311804adbc5f1a7fcff225f049R430

we should document that this will only work for coins whose metadata object is
either shared or frozen. This seems to be the case at least for all example
coins, so we can probably expect most coins to follow this pattern. Ones that
don't, however, will not be transferrable through the token bridge

## Creating new Coin Types

TODO: up to date implementation notes

Internally, `create_wrapped_coin` calls `coin::create_currency<CoinType>(witness, decimals, ctx)`, obtains a treasury cap, and finally stores
the treasury cap inside of a `TreasuryCapContainer` object, whose usage is restricted by the functions in its defining module (in particular, gated by VAAs). The `TreasuryCapContainer` is mutably shared so users can access it in a permissionless way. The reason that the treasury cap itself
is not mutably shared is that users would be able to use it to mint/burn tokens without limitation.

## Initiating and Completing Transfers

The Token Bridge stores both coins transferred by the user for lockup and treasury caps used for minting/burning wrapped assets. To this end, we implement two structs, which are both mutably shared and whose usage is restricted by VAA-gated functions defined in their parent modules.

```rust
struct TreasuryCapContainer<T> {
	t: TreasuryCap<T>,
}
```

```rust
struct CoinStore<T> {
	coins: coin<T>,
}
```

Accordingly, we define the following functions for initiating and completing transfers. There is a version of each for wrapped and native coins, because we can't store info about `CoinType` within `State`. There does not seem to be a way of introspecting the CoinType to determine whether it represents a native or wrapped asset. In addition, we have to use either a `TreasuryCapStore` or `CoinStore` depending on whether we want to initiate or complete a transfer for a native or wrapped asset, which leads to different function signatures.

### `complete_transfer_wrapped<T>(treasury_cap_store: &mut TreasuryCapStore<T>)`
- Use treasury cap to mint wrapped assets to recipient

### `complete_transfer_native<T>(store: &mut CoinStore<T>)`
- Idea is to extract coins from token_bridge and give them to the recipient. We pass in a mutably shared `CoinStore` object, which contains balance or coin objects belonging to token bridge. Coins are extracted from this object and passed to the recipient.

### `transfer_native<T>(coin: Coin<T>, store: &mut CoinStore<T>)`
- Transfer user-supplied native coins to `CoinStore`
### `transfer_wrapped<T>(treasury_cap_store: &mut TreasuryCapStore<T>)`
- Use the treasury cap to burn some user-supplied wrapped assets

## Contract Upgrades
Not yet supported in Sui.

## Bridge State
TODO: up to date implementation notes
