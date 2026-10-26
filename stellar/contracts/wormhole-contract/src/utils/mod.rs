//! Cryptographic and address conversion utilities.
//!
//! Provides helper functions for hashing, address format conversion between
//! Stellar and Ethereum/Wormhole formats, and token address resolution.

use soroban_sdk::{Address, Bytes, BytesN, Env, address_payload::AddressPayload};

/// Computes keccak256 hash of data, returning a 32-byte array.
pub fn keccak256_hash(env: &Env, data: &Bytes) -> BytesN<32> {
    env.crypto().keccak256(data).to_bytes()
}

/// Converts a 32-byte ED25519 public key to a Stellar `Address`.
///
/// Used for decoding fee transfer recipients from governance VAA payloads.
pub fn address_from_ed25519_pk_bytes(env: &Env, pk_bytes: &BytesN<32>) -> Address {
    Address::from_payload(
        env,
        AddressPayload::AccountIdPublicKeyEd25519(pk_bytes.clone()),
    )
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

/// Returns the native XLM Stellar Asset Contract address for this network.
pub fn get_native_token_address(env: &Env) -> Address {
    // `Asset::Native` XDR; the contract ID depends on the network ID.
    env.deployer()
        .with_stellar_asset([0u8; 4])
        .deployed_address()
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::testutils::Ledger;

    #[test]
    fn test_native_token_address_is_network_specific() {
        let env = Env::default();
        for (passphrase, expected) in [
            (
                "Test SDF Network ; September 2015",
                "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
            ),
            (
                "Public Global Stellar Network ; September 2015",
                "CAS3J7GYLGXMF6TDJBBYYSE3HQ6BBSMLNUQ34T6TZMYMW2EVH34XOWMA",
            ),
        ] {
            let network_id = env
                .crypto()
                .sha256(&Bytes::from_slice(&env, passphrase.as_bytes()));
            env.ledger().set_network_id(network_id.to_array());
            assert_eq!(
                get_native_token_address(&env),
                Address::from_str(&env, expected)
            );
        }
    }
}
