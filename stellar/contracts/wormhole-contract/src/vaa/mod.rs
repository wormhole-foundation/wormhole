//! VAA verification logic.
//!
//! This module contains contract-specific VAA verification that requires
//! access to storage (guardian sets). Parsing, serialization, and other
//! pure data operations are defined in the wormhole-interface crate.

pub use wormhole_soroban_client::{Signature, VAA};

use soroban_sdk::{Bytes, BytesN, Env, Vec};
use wormhole_soroban_client::WormholeError;

/// Verify a VAA's signatures against stored guardian sets.
///
/// This checks:
/// - Guardian set exists in storage
/// - Guardian set not expired
/// - Sufficient valid signatures (quorum)
/// - Signatures in ascending order
pub fn verify_vaa_signatures(vaa: &VAA, env: &Env) -> Result<bool, WormholeError> {
    // let body_bytes = vaa.serialize_body(env);

    // let guardian_set_info = governance::guardian_set::get(env, vaa.guardian_set_index)?;

    // if let Some(expiry) = governance::guardian_set::get_expiry(env, vaa.guardian_set_index) {
    //     if env.ledger().timestamp() > expiry {
    //         return Err(Error::GuardianSetExpired);
    //     }
    // }

    // verify_signatures_impl(vaa, env, &body_bytes, &guardian_set_info.keys)

    // TODO

    Err(WormholeError::GuardianSetNotFound)
}

/// Verify a single ECDSA signature against an expected Ethereum style address.
///
/// This performs:
/// - ECDSA signature recovery
/// - Public key to Ethereum style address conversion
/// - Address comparison
pub fn verify_signature(
    sig: &Signature,
    env: &Env,
    message_hash: &soroban_sdk::crypto::Hash<32>,
    expected_address: &BytesN<20>,
) -> Result<bool, WormholeError> {
    let mut sig_bytes = [0u8; 64];
    let r_array = sig.r.to_array();
    let s_array = sig.s.to_array();
    sig_bytes[..32].copy_from_slice(&r_array);
    sig_bytes[32..].copy_from_slice(&s_array);
    let sig_formatted = BytesN::<64>::from_array(env, &sig_bytes);

    let recovered_pubkey = env
        .crypto()
        .secp256k1_recover(message_hash, &sig_formatted, sig.v);

    let eth_address = crate::utils::pubkey_to_eth_address(env, &recovered_pubkey);

    Ok(&eth_address == expected_address)
}

/// Calculate the required quorum for a given number of guardians.
/// Formula: (num_guardians * 2 / 3) + 1
fn calculate_quorum(num_guardians: u32) -> u32 {
    num_guardians
        .saturating_mul(2)
        .saturating_div(3)
        .saturating_add(1)
}

/// Verify all signatures against a specific guardian set
fn verify_signatures_impl(
    vaa: &VAA,
    env: &Env,
    body_bytes: &Bytes,
    guardian_keys: &Vec<BytesN<20>>,
) -> Result<bool, WormholeError> {
    let guardian_count = guardian_keys.len();
    if guardian_count == 0 {
        return Err(WormholeError::InvalidGuardianSetIndex);
    }

    let required_sigs = calculate_quorum(guardian_count);
    if vaa.signatures.len() < required_sigs {
        return Err(WormholeError::InsufficientSignatures);
    }

    let body_hash_bytes: Bytes = crate::utils::keccak256_hash(env, body_bytes).into();
    let double_hash = env.crypto().keccak256(&body_hash_bytes);

    let mut last_guardian_index = None;

    for signature in vaa.signatures.iter() {
        if let Some(last_idx) = last_guardian_index {
            if signature.guardian_index <= last_idx {
                return Err(WormholeError::SignaturesNotAscending);
            }
        }
        last_guardian_index = Some(signature.guardian_index);

        if signature.guardian_index >= guardian_count {
            return Err(WormholeError::GuardianIndexOutOfBounds);
        }

        let guardian_key = guardian_keys
            .get(signature.guardian_index)
            .ok_or(WormholeError::GuardianIndexOutOfBounds)?;

        if !verify_signature(&signature, env, &double_hash, &guardian_key)? {
            return Err(WormholeError::InvalidSignature);
        }
    }

    Ok(true)
}

/// Parse and verify a VAA from bytes.
pub fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<bool, WormholeError> {
    let vaa = VAA::try_from((&env, &vaa_bytes))?;
    verify_vaa_signatures(&vaa, &env)
}

/// Parse a VAA from bytes.
pub fn parse_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<VAA, WormholeError> {
    VAA::try_from((env, vaa_bytes))
}
