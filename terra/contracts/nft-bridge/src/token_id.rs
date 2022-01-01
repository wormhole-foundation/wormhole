use bigint::U256;
use cosmwasm_std::{
    StdError,
    StdResult,
    Storage,
};

use sha3::{
    Digest,
    Keccak256,
};
use wormhole::byte_utils::ByteUtils;

use crate::{
    state::{
        token_id_hashes,
        token_id_hashes_read,
    },
    CHAIN_ID,
};

// NOTE: [External and internal token id conversion]
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
//
// The CW721 NFT standard allows token ids to be arbitrarily long (utf8)
// strings, while the token_ids in VAA payloads are always 32 bytes (and not
// necessarily valid utf8).
//
// We call a token id that's in string format an "internal id", and a token id
// that's in 32 byte format an "external id". Note that whether a token id is in
// internal format or external format doesn't imply which chain the token id
// originates from. We can have a terra (native) token id in both internal and
// external formats, and likewise we can have an ethereum token in in both
// internal and external formats.
//
// To support seamless transfers through the bridge, we need a way to have a
// 1-to-1 mapping from internal ids to external ids.
// When a foreign (such as ethereum or solana) token id first comes through, we
// simply render it into a string by formatting it as a decimal number. Then,
// when we want to transfer such a token back through the bridge, we simply
// parse the string back into a u256 (32 byte) number.
//
// When a native token id first leaves through the bridge, we turn its id into a
// 32 byte hash (keccak256). This hash is the external id. We store a mapping
//
//    (chain_id, nft_address, keccak256(internal_id)) => internal_id
//
// so that we can turn it back into an internal id when it comes back through
// the bridge. When the token is sent back, we could choose to delete the hash
// from the store, but we do not. This way, external token verifiers will be
// able to verify NFT origins even for NFTs that have been transferred back.
//
// If two token ids within the same contract have the same keccak256 hash, then
// it's possible to lose tokens, but this is very unlikely.

pub fn from_external_token_id(
    storage: &mut dyn Storage,
    nft_chain: u16,
    nft_address: &[u8; 32],
    token_id_external: &[u8; 32],
) -> StdResult<String> {
    if nft_chain == CHAIN_ID {
        token_id_hashes_read(storage, nft_chain, *nft_address).load(token_id_external)
    } else {
        Ok(format!("{}", U256::from_big_endian(token_id_external)))
    }
}

fn hash(token_id: &String) -> Vec<u8> {
    let mut hasher = Keccak256::new();
    hasher.update(token_id);
    hasher.finalize().to_vec()
}

pub fn to_external_token_id(
    storage: &mut dyn Storage,
    nft_chain: u16,
    nft_address: &[u8; 32],
    token_id_internal: String,
) -> StdResult<[u8; 32]> {
    if nft_chain == CHAIN_ID {
        let hash = hash(&token_id_internal);
        token_id_hashes(storage, nft_chain, *nft_address).save(&hash, &token_id_internal)?;
        Ok(hash.as_slice().get_const_bytes(0))
    } else {
        let mut bytes = [0; 32];
        U256::from_dec_str(&token_id_internal)
            .map_err(|_| {
                StdError::generic_err(format!(
                    "{} could not be parsed as a decimal number",
                    token_id_internal
                ))
            })?
            .to_big_endian(&mut bytes);
        Ok(bytes)
    }
}
