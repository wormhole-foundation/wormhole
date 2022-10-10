# Sui Wormhole Token Bridge Design

The Token Bridge is responsible for storing treasury caps and locked tokens and exposing functions for initiating and completing transfers, which are gated behind VAAs. It also supports token attestations from foreign chains (which must be done once prior to transfer), contract upgrades, and chain registration.

## Token Attestation

Right now it is unclear how to do token attestation.

There doesn't seem to be a standardized way to store info about a coin. One way is to create a `CoinInfo` object containing the name, symbol, decimals of the token, and `transfer::freeze_object` it.

## Creating new Coin Types
We emulate the transferable witness pattern to allow users to call into the token bridge contract and prompt it to create a new currency representing a wrapped asset.

A user should first get a token attestation VAA, copy it into the one-time-witness coin template outlined below, and finally publish the package so that
`init` is run and `create_wrapped_coin` is called with the one-time-witness `COIN_WITNESS`.

```rust
 // === coin_witness.move ===

 use token_bridge::wrapped::create_wrapped_coin;

 //one-time witness definition
 struct COIN_WITNESS has store, drop {}

    // publish this module to call into token_bridge and create a wrapped coin
    fun init(_ctx: &mut TxContext) {
        // paste the attestation VAA below
        let _vaa = x"deadbeef00001231231231";

        create_wrapped_coin(vaa, COIN_WITNESS {})
    }
```

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

Accordingly, we define the following functions for initiating and completing transfers. There is a version of each for wrapped and native coins, because we can't store info about `CoinType` within `BridgeState`. There does not seem to be a way of introspecting the CoinType to determine whether it represents a native or wrapped asset. In addition, we have to use either a `TreasuryCapStore` or `CoinStore` depending on whether we want to initiate or complete a transfer for a native or wrapped asset, which leads to different function signatures.

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
This object contains the set `consumed_vaas`, the hashmap `treasury_cap_stores: VecMap<OriginInfo, &UID>`, `governance_chain_id`, and `governance_contract`. The current bridge state looks like the following.

```rust
  struct BridgeState {
      governance_chain_id: U16,
      governance_contract: ExternalAddress,

      /// Set of consumed VAA hashes
      consumed_vaas: VecSet<vector<u8>>,

      /// Track treasury caps IDs, which are mutably shared
      treasury_cap_stores: VecMap<OriginInfo, &UID>,

      // Mapping of bridge contracts on other chains
      registered_emitters: VecMap<U16, ExternalAddress>,
   }
```
