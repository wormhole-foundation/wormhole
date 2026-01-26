//! VAA verification and signature validation.
//!
//! Provides cryptographic verification of VAA signatures against stored guardian sets.
//! Uses secp256k1 ECDSA recovery to derive Ethereum addresses from signatures, then
//! compares against the known guardian keys.
//!
//! Parsing and serialization are in `wormhole-soroban-client::types`.

pub use wormhole_soroban_client::{Signature, VAA};

use crate::governance;
use soroban_sdk::{Bytes, BytesN, Env, Vec};
use wormhole_soroban_client::WormholeError;

/// Verifies all VAA signatures against the stored guardian set.
///
/// Validates guardian set existence and expiry, then checks that:
/// - At least quorum signatures are present (13/19 for mainnet)
/// - Signature indices are strictly ascending (prevents reuse)
/// - Each signature recovers to the expected guardian's Ethereum address
///
/// Uses double keccak256 hashing as per Wormhole spec: `keccak256(keccak256(body))`.
pub fn verify_vaa_signatures(vaa: &VAA, env: &Env) -> Result<bool, WormholeError> {
    let body_bytes = vaa.serialize_body(env);

    let guardian_set_info = governance::guardian_set::get_guardian_set(env, vaa.guardian_set_index)?;

    if let Some(expiry) = governance::guardian_set::get_guardian_set_expiry(env, vaa.guardian_set_index) {
        if env.ledger().timestamp() > expiry {
            return Err(WormholeError::GuardianSetExpired);
        }
    }

    verify_signatures_impl(vaa, env, &body_bytes, &guardian_set_info.keys)
}

/// Verifies a single ECDSA signature matches an expected Ethereum address.
///
/// Recovers the secp256k1 public key from the signature and message hash,
/// derives the Ethereum address (last 20 bytes of keccak256(pubkey[1..])),
/// and compares against the expected guardian address.
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

/// Calculates required quorum: `(n * 2 / 3) + 1` (e.g., 13 of 19 guardians).
fn calculate_quorum(num_guardians: u32) -> u32 {
    num_guardians
        .saturating_mul(2)
        .saturating_div(3)
        .saturating_add(1)
}

/// Verifies all signatures against a guardian key set using double keccak256.
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

/// Parses and verifies a VAA in a single call.
///
/// Combines `VAA::try_from` parsing with full signature verification.
pub fn verify_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<bool, WormholeError> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_vaa_signatures(&vaa, &env)
}

/// Parses VAA bytes into a structured `VAA` without signature verification.
pub fn parse_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<VAA, WormholeError> {
    VAA::try_from((env, vaa_bytes))
}
