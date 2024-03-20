# Aptos NFT Bridge

This contract is a reference implementation of the [Wormhole NFT bridge
specification](../../whitepapers/0006_nft_bridge.md) on Aptos, written in the
Move programming language.

This document provides an overview of the design and structure of the program.

## NFTs on Aptos

The [Aptos Token
specification](https://aptos.dev/concepts/coin-and-token/aptos-token/) provides
a good overview of how Tokens are specified on Aptos, but we review the relevant
parts here.

First, it's important to mention that the Token specification is more general
than NFTs, as Tokens can be used to describe both fungible and non-fungible
tokens.

Tokens belong to collections, which in turn belong to their creators. Given a
creator address, the collection's name (string) uniquely identifies the
collection. Within a collection, a token's name uniquely identifies the token.

These tokens may be fungible however: it's possible to have multiple copies of
them which are fully interchangeable. Each type of token has a set of properties
of key-value pairs, that allow the creator to attach custom information to the
tokens, such as hair colour. For an example, see
https://www.topaz.so/assets/Aptos-Undead-5a4505c2e9/Aptos%20Undead%20%233085/0.

The base token (which is identified by `(creator, collection_name, token_name)`)
has a set of "default" properties. It is possible to create "editions" of these
base tokens, which turn them into unique variations with additional properties
relative to the base token. The modified editions are unique and non-fungible,
so they are NFTs. When such an edition is created, it gets assigned a version,
called the `property_version`, within the base token. Such editions are thus
identified by `(creator, collection_name, token_name, property_version)`. When
the `property_version` is 0, the token may be fungible, but when it is non-0,
only a single copy may exist.

From the documentation, it appears that this property versioning is more of an
optimisation to allow bulk-minting NFTs cheaply and later add properties in a
copy-on-write fashion. It's unclear if the fungibility is meaningful outside of
this optimisation, as fungible tokens are already better supported by the first
class `Coin` type.

## Wormhole NFT Bridge

The Wormhole NFT bridge specification (which is based on ERC721) uses 32 bytes
to identify collections (only 20 bytes of which are used on EVM chains, for the
contract address) and another 32 bytes to identify the token within the
collection. Neither of these fields are sufficient to pack the necessary
information on Aptos, since the creator address itself is already 32 bytes, and
the collection name can be an arbitrary string up to 128 bytes. Token names can
also be arbitrary 128 byte strings.

Thus, we store 32 byte hashes of these two fields respectively. The exact
details of how the hashes are computed are defined in
[token_hash.move](./sources/token_hash.move). The collection's hash is computed
from the creator and the collection name. The hash of the individual NFTs is
computed from the creator, the collection's name, the token name, and the
property version. Note that it would be sufficient to just take the token name
and the property version, but this way the token's hash can be used as a
globally unique identifier, which simplifies the implementation.

When transferring a native token out for the first time, we
(`state::set_native_asset_info`) store a mapping from its hash to the token's
`TokenId`, so it can be retrieved when transferring the token back
(`state::get_native_asset_info`).

### Wrapped asset creation

When transferring an NFT from a collection on a foreign chain to Aptos, a
corresponding "wrapped" collection is created. The module responsible for this
is [wrapped.move](sources/wrapped.move). The collection name is the the NFT name
field from the transfer VAA. To avoid collisions here, each NFT is minted into a
freshly created creator account, implemented as a resource account.

### Handling "fungible" tokens

As discussed above, tokens whose property version is 0 are technically fungible.
We could disallow tokens whose property version is 0, and only allow
transferring ones that are non-0. However, many real-world NFT projects (such as
[Aptos Undead](https://www.topaz.so/collection/Aptos-Undead-5a4505c2e9)) simply
mint all tokens as separate tokens with property version 0 (and don't
necessarily use editions). We could instead check that the supply of the token
is 1, but new tokens can always be minted after the check is performed anyway.
Also, the supply is only tracked if the token has a specified maximum supply,
which, again, real-world NFT projects may not specify.

Instead, we don't check the supply, and simply allow transferring a single copy
of `property_version = 0` tokens at a given time. What this means is that when a
token is transferred out, we check that only a single copy is sent at a given
time, and also that there is at most 1 token held by the NFT bridge contract.
This is the most general setup that supports existing NFT projects, but it does
mean there is an edge case where tokens that are legitimately fungible (but
decided to not use the `Coin` type for some reason) are transferrable through
the NFT bridge, although at most 1 can be locked at any given time, so this edge
case is not observable outside of Aptos.

An additional caveat: it is possible for the creator to mutate the properties of
an NFT by calling `token::mutate_one_token` (in fact this is the mechanism by
which property versions other than 0 are assigned). If the token already had a
non-0 property version, then this operation will simply mutate it in-place,
keeping the identity of the token. However, if the property version was 0, then
the token is burned and a new token with a non-0 property version is created in
its place. If this happens to a token held in custody by the NFT bridge, then
that token will be irredeemable. It does require the creator of the NFT to
explicitly mutate a token held by the NFT bridge.

### Handling Solana NFTs

Solana NFTs require special handling currently. This is because at the time the
Solana NFT bridge was first implemented, there was no notion of NFT collections,
and each NFT would simply be its own individual token. Due to the gas costs of
creating collections on Ethereum, the Solana NFT bridge simply puts all NFTs
into a single dummy collection, so when transferred to other chains, they end up
under the same collection. This means that storing the collection metadata in
the wrapped collection does not work due to the many-to-one mapping. Instead,
like on Ethereum, we implement a cache (the "SPL cache") to store the name and
symbol of these tokens separately in a mapping keyed by the solana token's
address. When transferring out, this cache is consulted to recover the metadata
needed in the outgoing VAA.

The `state::is_unified_solana_collection` implements the check to determine
whether this caching behaviour is needed. It not only checks for the source
chain (Solana) but also the dummy collection address. This allows smoothly
upgrading the Solana NFT bridge to use the real collection address, in which
case the collections will be preserved moving forward and the cache ignored.

The cache is set in `wrapped::create_or_find_wrapped_nft_collection` when
transferring in, and read by the `state::get_wrapped_asset_name_and_symbol`
function (used by `transfer_nft::lock_or_burn` on the way out).

## Governance

Outside of handling NFT transfers, the NFT bridge can perform two additional
operations, both of which require a VAA signed by the Wormhole guardians.
These are governance operations, as they alter the behaviour of the bridge.
Both of these governance operations are identical to the token bridge implementation.

### Registrations

Since sending messages through Wormhole is permissionless and message payloads
are arbitrary, any program could send messages that look like NFT transfers.
To ensure that such messages are accepted from a trusted set of contracts, the
NFT bridge maintains a set of known "emitters". These are stored in a table
`registered_emitters` in `state::State`, keyed by the chains' ids (i.e. at most
one emitter per chain). This mapping can be updated by submitting registration
VAAs (which are special VAAs that are signed manually by the guardians through a
governance ceremony), and handled in the `register_chain.move` module.

### Contract upgrades

Contract upgrades also require governance VAAs. In the case of Aptos, the VAA
will contain the hash of the bytecode we're upgrading to. This logic is
implemented in `contract_upgrade.move`.
