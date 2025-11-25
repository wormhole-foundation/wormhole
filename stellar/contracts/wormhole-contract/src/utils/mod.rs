mod bytes_reader;

pub(crate) use bytes_reader::BytesReader;

use crate::constants::NATIVE_TOKEN_ADDRESS;
use soroban_sdk::{Address, Bytes, BytesN, Env, String};
use stellar_strkey::{Strkey, ed25519::PublicKey as StrEd25519};

/// Compute keccak256 hash and return as BytesN<32>.
///
/// This is a convenience wrapper around `env.crypto().keccak256().to_bytes()`
/// for cases where the hash is needed as BytesN rather than Hash type.
pub(crate) fn keccak256_hash(env: &Env, data: &Bytes) -> BytesN<32> {
    env.crypto().keccak256(data).to_bytes()
}

/// Convert 32-byte ED25519 public key to Stellar Address
pub(crate) fn address_from_ed25519_pk_bytes(env: &Env, pk_bytes: &BytesN<32>) -> Address {
    let str_pk = Strkey::PublicKeyEd25519(StrEd25519(pk_bytes.to_array()));
    Address::from_string(&String::from_str(env, &str_pk.to_string()))
}

/// Convert a Stellar address to 32-byte representation (keccak hash of StrKey)
pub(crate) fn address_to_bytes32(env: &Env, address: &Address) -> BytesN<32> {
    let addr_string = address.to_string();
    let addr_bytes_obj = addr_string.to_bytes();
    keccak256_hash(env, &addr_bytes_obj)
}

/// Convert a recovered secp256k1 public key to an Ethereum style address
pub(crate) fn pubkey_to_eth_address(env: &Env, pubkey: &BytesN<65>) -> BytesN<20> {
    let pubkey_array = pubkey.to_array();
    let pubkey_bytes = Bytes::from_array(env, &pubkey_array);
    let pubkey_without_prefix = pubkey_bytes.slice(1..);

    let hash = env.crypto().keccak256(&pubkey_without_prefix);

    let hash_array = hash.to_array();
    let mut addr_bytes = [0u8; 20];
    addr_bytes.copy_from_slice(&hash_array[12..32]);

    BytesN::from_array(env, &addr_bytes)
}

/// Get native XLM token address
pub(crate) fn get_native_token_address(env: &Env) -> Address {
    Address::from_string(&String::from_str(env, NATIVE_TOKEN_ADDRESS))
}
