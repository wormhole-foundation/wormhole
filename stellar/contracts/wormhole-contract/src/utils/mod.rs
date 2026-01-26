//! Cryptographic and address conversion utilities.
//!
//! Provides helper functions for hashing, address format conversion between
//! Stellar and Ethereum/Wormhole formats, and token address resolution.

use soroban_sdk::{Address, Bytes, BytesN, Env, String};
use stellar_strkey::{Strkey, ed25519::PublicKey as StrEd25519};
use wormhole_soroban_client::NATIVE_TOKEN_ADDRESS;

/// Computes keccak256 hash of data, returning a 32-byte array.
pub fn keccak256_hash(env: &Env, data: &Bytes) -> BytesN<32> {
    env.crypto().keccak256(data).to_bytes()
}

/// Converts a 32-byte ED25519 public key to a Stellar `Address`.
///
/// Used for decoding fee transfer recipients from governance VAA payloads.
pub fn address_from_ed25519_pk_bytes(env: &Env, pk_bytes: &BytesN<32>) -> Address {
    let str_pk = Strkey::PublicKeyEd25519(StrEd25519(pk_bytes.to_array()));
    Address::from_string(&String::from_str(env, &str_pk.to_string()))
}

/// Converts a Stellar address to its 32-byte Wormhole representation.
///
/// Computes `keccak256(strkey)` to produce a deterministic 32-byte identifier
/// compatible with the Wormhole message format.
pub fn address_to_bytes32(env: &Env, address: &Address) -> BytesN<32> {
    let addr_string = address.to_string();
    let addr_bytes_obj = addr_string.to_bytes();
    keccak256_hash(env, &addr_bytes_obj)
}

/// Derives an Ethereum address from a recovered secp256k1 public key.
///
/// Strips the 0x04 prefix, hashes the remaining 64 bytes with keccak256,
/// and takes the last 20 bytes as the Ethereum address.
pub fn pubkey_to_eth_address(env: &Env, pubkey: &BytesN<65>) -> BytesN<20> {
    let pubkey_array = pubkey.to_array();
    let pubkey_bytes = Bytes::from_array(env, &pubkey_array);
    let pubkey_without_prefix = pubkey_bytes.slice(1..);

    let hash = env.crypto().keccak256(&pubkey_without_prefix);

    let hash_array = hash.to_array();
    let mut addr_bytes = [0u8; 20];
    addr_bytes.copy_from_slice(&hash_array[12..32]);

    BytesN::from_array(env, &addr_bytes)
}

/// Returns the native XLM Stellar Asset Contract address.
pub fn get_native_token_address(env: &Env) -> Address {
    Address::from_string(&String::from_str(env, NATIVE_TOKEN_ADDRESS))
}
