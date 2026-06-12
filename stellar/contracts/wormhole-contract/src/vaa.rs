//! VAA verification and signature validation.
//!
//! Provides cryptographic verification of VAA signatures against stored
//! guardian sets. Uses secp256k1 ECDSA recovery to derive Ethereum addresses
//! from signatures, then compares against the known guardian keys.
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
/// Uses double keccak256 hashing as per Wormhole spec:
/// `keccak256(keccak256(body))`.
pub fn verify_vaa_signatures(vaa: &VAA, env: &Env) -> Result<bool, WormholeError> {
    let body_bytes = vaa.serialize_body(env);

    let guardian_set_info =
        governance::guardian_set::get_guardian_set(env, vaa.guardian_set_index)?;

    if let Some(expiry) =
        governance::guardian_set::get_guardian_set_expiry(env, vaa.guardian_set_index)
        && env.ledger().timestamp() > expiry
    {
        return Err(WormholeError::GuardianSetExpired);
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
        if let Some(last_idx) = last_guardian_index
            && signature.guardian_index <= last_idx
        {
            return Err(WormholeError::SignaturesNotAscending);
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

/// Parses a VAA and verifies all guardian signatures in a single call.
///
/// This is the primary entry point for external integrators who need both
/// parsing and verification. Avoids the double parse that occurs when
/// calling `verify_vaa` and `parse_vaa` separately.
pub fn parse_and_verify_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<VAA, WormholeError> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_vaa_signatures(&vaa, env)?;
    Ok(vaa)
}

/// Verifies a VAA's guardian signatures, returning success or an error.
///
/// Parses and verifies internally. For callers that also need the parsed
/// VAA, use [`parse_and_verify_vaa`] instead to avoid a double parse.
pub fn verify_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<(), WormholeError> {
    parse_and_verify_vaa(env, vaa_bytes)?;
    Ok(())
}

/// Parses VAA bytes into a structured `VAA` without signature verification.
pub fn parse_vaa(env: &Env, vaa_bytes: &Bytes) -> Result<VAA, WormholeError> {
    VAA::try_from((env, vaa_bytes))
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::{testutils::Ledger, vec};
    use wormhole_soroban_client::{ConsistencyLevel, GOVERNANCE_EMITTER};

    fn deploy_initialized(env: &Env) -> (soroban_sdk::Address, crate::WormholeClient<'_>) {
        let guardian = BytesN::<20>::from_array(env, &[0u8; 20]);
        let initial_guardians = vec![env, guardian];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(crate::Wormhole, (initial_guardians, governance_emitter));
        let client = crate::WormholeClient::new(env, &contract_id);
        (contract_id, client)
    }

    fn base_vaa_with_guardian_set(env: &Env, guardian_set_index: u32) -> VAA {
        VAA {
            version: 1,
            guardian_set_index,
            signatures: Vec::new(env),
            timestamp: 0x01020304,
            nonce: 0xA0B0C0D0,
            emitter_chain: 61,
            emitter_address: BytesN::<32>::from_array(env, &[0x33u8; 32]),
            sequence: 7,
            consistency_level: ConsistencyLevel::Confirmed,
            payload: Bytes::from_slice(env, &[0xAA, 0xBB]),
        }
    }

    fn sign_for_vaa(
        env: &Env,
        vaa: &VAA,
        sk_bytes: [u8; 32],
        guardian_index: u32,
    ) -> (Signature, BytesN<20>) {
        let body_bytes = vaa.serialize_body(env);
        let body_hash_bytes: Bytes = crate::utils::keccak256_hash(env, &body_bytes).into();
        let double_hash = env.crypto().keccak256(&body_hash_bytes);
        let msg32: [u8; 32] = double_hash.to_array();

        let secp = secp256k1::Secp256k1::new();
        let sk = secp256k1::SecretKey::from_byte_array(sk_bytes).unwrap();
        let pk = secp256k1::PublicKey::from_secret_key(&secp, &sk);
        let pk_bytes65 = BytesN::<65>::from_array(env, &pk.serialize_uncompressed());
        let guardian_eth = crate::utils::pubkey_to_eth_address(env, &pk_bytes65);

        let message = secp256k1::Message::from_digest(msg32);
        let sig = secp.sign_ecdsa_recoverable(message, &sk);
        let (recid, compact64) = sig.serialize_compact();

        let mut r_arr = [0u8; 32];
        let mut s_arr = [0u8; 32];
        r_arr.copy_from_slice(&compact64[0..32]);
        s_arr.copy_from_slice(&compact64[32..64]);

        (
            Signature {
                guardian_index,
                r: BytesN::<32>::from_array(env, &r_arr),
                s: BytesN::<32>::from_array(env, &s_arr),
                v: i32::from(recid) as u32,
            },
            guardian_eth,
        )
    }

    #[test]
    fn test_parse_vaa_valid() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);

        // Known fields
        let version_u8: u8 = 1;
        let guardian_set_index: u32 = 5;
        let num_signatures_u8: u8 = 0;

        let timestamp: u32 = 0x01020304;
        let nonce: u32 = 0xA0B0C0D0;
        let emitter_chain_u16: u16 = 61;
        let emitter_address_arr: [u8; 32] = [
            0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D,
            0x0E, 0x0F, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B,
            0x1C, 0x1D, 0x1E, 0x1F,
        ];
        let sequence: u64 = 0x0102030405060708;
        let consistency_level_u8: u8 = ConsistencyLevel::Confirmed as u8;
        let payload: [u8; 2] = [0xAA, 0xBB];

        let mut raw = Bytes::new(&env);

        raw.append(&Bytes::from_slice(&env, &[version_u8]));
        raw.append(&Bytes::from_slice(&env, &guardian_set_index.to_be_bytes()));
        raw.append(&Bytes::from_slice(&env, &[num_signatures_u8]));
        // signatures: none

        raw.append(&Bytes::from_slice(&env, &timestamp.to_be_bytes()));
        raw.append(&Bytes::from_slice(&env, &nonce.to_be_bytes()));
        raw.append(&Bytes::from_slice(&env, &emitter_chain_u16.to_be_bytes()));
        raw.append(&Bytes::from_slice(&env, &emitter_address_arr));
        raw.append(&Bytes::from_slice(&env, &sequence.to_be_bytes()));
        raw.append(&Bytes::from_slice(&env, &[consistency_level_u8]));
        raw.append(&Bytes::from_slice(&env, &payload));

        let vaa = client.parse_vaa(&raw);

        assert_eq!(vaa.version, u32::from(version_u8));
        assert_eq!(vaa.guardian_set_index, guardian_set_index);
        assert_eq!(vaa.signatures.len(), 0);

        assert_eq!(vaa.timestamp, timestamp);
        assert_eq!(vaa.nonce, nonce);
        assert_eq!(vaa.emitter_chain, u32::from(emitter_chain_u16));

        let expected_emitter_address = BytesN::<32>::from_array(&env, &emitter_address_arr);
        assert_eq!(vaa.emitter_address, expected_emitter_address);

        assert_eq!(vaa.sequence, sequence);
        assert_eq!(vaa.consistency_level, ConsistencyLevel::Confirmed);

        let expected_payload = Bytes::from_slice(&env, &payload);
        assert_eq!(vaa.payload, expected_payload);
    }

    #[test]
    fn test_parse_vaa_invalid_format() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);

        let b0 = Bytes::new(&env);
        let r0 = client.try_parse_vaa(&b0);
        assert_eq!(r0, Err(Ok(WormholeError::InvalidVAAFormat)));

        let b1 = Bytes::from_slice(&env, &[0u8; 56]);
        let r1 = client.try_parse_vaa(&b1);
        assert_eq!(r1, Err(Ok(WormholeError::InvalidVAAFormat)));

        let mut b2 = Bytes::new(&env);

        // header
        b2.append(&Bytes::from_slice(&env, &[1u8])); // version
        b2.append(&Bytes::from_slice(&env, &0u32.to_be_bytes())); // guardian_set_index
        b2.append(&Bytes::from_slice(&env, &[1u8])); // num_signatures = 1

        // body (51 bytes), payload empty
        b2.append(&Bytes::from_slice(&env, &0u32.to_be_bytes())); // timestamp
        b2.append(&Bytes::from_slice(&env, &0u32.to_be_bytes())); // nonce
        b2.append(&Bytes::from_slice(&env, &0u16.to_be_bytes())); // emitter_chain
        b2.append(&Bytes::from_slice(&env, &[0u8; 32])); // emitter_address
        b2.append(&Bytes::from_slice(&env, &0u64.to_be_bytes())); // sequence
        b2.append(&Bytes::from_slice(&env, &[0u8])); // consistency_level

        let r2 = client.try_parse_vaa(&b2);
        assert_eq!(r2, Err(Ok(WormholeError::InvalidVAAFormat)));
    }

    #[test]
    fn test_get_body_bytes_function() {
        let env = Env::default();

        let mut raw = Bytes::new(&env);

        // header (6 bytes)
        raw.append(&Bytes::from_slice(&env, &[1u8])); // version
        raw.append(&Bytes::from_slice(&env, &5u32.to_be_bytes())); // guardian_set_index
        raw.append(&Bytes::from_slice(&env, &[0u8])); // sig_count = 0

        // body (51 + payload)
        raw.append(&Bytes::from_slice(&env, &0x01020304u32.to_be_bytes())); // timestamp
        raw.append(&Bytes::from_slice(&env, &0xA0B0C0D0u32.to_be_bytes())); // nonce
        raw.append(&Bytes::from_slice(&env, &61u16.to_be_bytes())); // emitter_chain
        raw.append(&Bytes::from_slice(&env, &[0x11u8; 32])); // emitter_address
        raw.append(&Bytes::from_slice(
            &env,
            &0x0102030405060708u64.to_be_bytes(),
        )); // sequence
        raw.append(&Bytes::from_slice(
            &env,
            &[ConsistencyLevel::Confirmed as u8],
        ));
        raw.append(&Bytes::from_slice(&env, &[0xAAu8, 0xBBu8, 0xCCu8])); // payload

        let body = VAA::get_body_bytes(&raw).expect("expected valid body bytes");

        let mut expected_body = Bytes::new(&env);
        expected_body.append(&Bytes::from_slice(&env, &0x01020304u32.to_be_bytes())); // timestamp
        expected_body.append(&Bytes::from_slice(&env, &0xA0B0C0D0u32.to_be_bytes())); // nonce
        expected_body.append(&Bytes::from_slice(&env, &61u16.to_be_bytes())); // emitter_chain
        expected_body.append(&Bytes::from_slice(&env, &[0x11u8; 32])); // emitter_address
        expected_body.append(&Bytes::from_slice(
            &env,
            &0x0102030405060708u64.to_be_bytes(),
        )); // sequence
        expected_body.append(&Bytes::from_slice(
            &env,
            &[ConsistencyLevel::Confirmed as u8],
        ));
        expected_body.append(&Bytes::from_slice(&env, &[0xAAu8, 0xBBu8, 0xCCu8])); // payload

        assert_eq!(body, expected_body);

        let too_short = Bytes::from_slice(&env, &[1u8, 2u8, 3u8, 4u8, 5u8]);
        let r0 = VAA::get_body_bytes(&too_short);
        assert_eq!(r0, Err(WormholeError::InvalidVAAFormat));

        let mut corrupt = Bytes::new(&env);

        corrupt.append(&Bytes::from_slice(&env, &[1u8]));
        corrupt.append(&Bytes::from_slice(&env, &0u32.to_be_bytes()));
        corrupt.append(&Bytes::from_slice(&env, &[1u8]));

        // body (51 bytes), payload empty
        corrupt.append(&Bytes::from_slice(&env, &0u32.to_be_bytes())); // timestamp
        corrupt.append(&Bytes::from_slice(&env, &0u32.to_be_bytes())); // nonce
        corrupt.append(&Bytes::from_slice(&env, &0u16.to_be_bytes())); // emitter_chain
        corrupt.append(&Bytes::from_slice(&env, &[0u8; 32])); // emitter_address
        corrupt.append(&Bytes::from_slice(&env, &0u64.to_be_bytes())); // sequence
        corrupt.append(&Bytes::from_slice(&env, &[0u8])); // consistency_level

        let r1 = VAA::get_body_bytes(&corrupt);
        assert_eq!(r1, Err(WormholeError::InvalidVAAFormat));
    }

    #[test]
    fn test_verify_vaa_insufficient_sigs() {
        let env = Env::default();
        let governance_emitter = BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER);

        let g0 = BytesN::<20>::from_array(&env, &[0u8; 20]);
        let g1 = BytesN::<20>::from_array(&env, &[1u8; 20]);
        let g2 = BytesN::<20>::from_array(&env, &[2u8; 20]);
        let initial_guardians = vec![&env, g0, g1, g2];
        let contract_id = env.register(crate::Wormhole, (initial_guardians, governance_emitter));

        let mut sigs = Vec::<Signature>::new(&env);
        sigs.push_back(Signature {
            guardian_index: 0,
            r: BytesN::<32>::from_array(&env, &[0u8; 32]),
            s: BytesN::<32>::from_array(&env, &[0u8; 32]),
            v: 0,
        });

        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: sigs,
            timestamp: 0,
            nonce: 0,
            emitter_chain: 61,
            emitter_address: BytesN::<32>::from_array(&env, &[0x11u8; 32]),
            sequence: 0,
            consistency_level: ConsistencyLevel::Confirmed,
            payload: Bytes::from_slice(&env, &[0xAA, 0xBB]),
        };

        let res = env.as_contract(&contract_id, || verify_vaa_signatures(&vaa, &env));
        assert_eq!(res, Err(WormholeError::InsufficientSignatures));
    }

    #[test]
    fn test_verify_vaa_invalid_signature() {
        let env = Env::default();
        let stored_guardian = BytesN::<20>::from_array(&env, &[0x11u8; 20]);
        let initial_guardians = soroban_sdk::vec![&env, stored_guardian];
        let governance_emitter = BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(crate::Wormhole, (initial_guardians, governance_emitter));

        let base_vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: Vec::new(&env),
            timestamp: 0x01020304,
            nonce: 0xA0B0C0D0,
            emitter_chain: 61,
            emitter_address: BytesN::<32>::from_array(&env, &[0x33u8; 32]),
            sequence: 7,
            consistency_level: ConsistencyLevel::Confirmed,
            payload: Bytes::from_slice(&env, &[0xAA, 0xBB]),
        };

        let body_bytes = base_vaa.serialize_body(&env);
        let body_hash_bytes: Bytes = crate::utils::keccak256_hash(&env, &body_bytes).into();
        let double_hash = env.crypto().keccak256(&body_hash_bytes);
        let msg32: [u8; 32] = double_hash.to_array();

        let secp = secp256k1::Secp256k1::new();

        let sk = secp256k1::SecretKey::from_byte_array([0x42u8; 32]).unwrap();

        let message = secp256k1::Message::from_digest(msg32);

        let sig = secp.sign_ecdsa_recoverable(message, &sk);
        let (recid, compact64) = sig.serialize_compact();

        let mut r_arr = [0u8; 32];
        let mut s_arr = [0u8; 32];
        r_arr.copy_from_slice(&compact64[0..32]);
        s_arr.copy_from_slice(&compact64[32..64]);

        let sig_entry = Signature {
            guardian_index: 0,
            r: BytesN::<32>::from_array(&env, &r_arr),
            s: BytesN::<32>::from_array(&env, &s_arr),
            v: i32::from(recid) as u32,
        };

        let mut sigs = Vec::new(&env);
        sigs.push_back(sig_entry);

        let mut vaa = base_vaa.clone();
        vaa.signatures = sigs;

        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Err(WormholeError::InvalidSignature));
        });
    }

    #[test]
    fn test_verify_vaa_valid_signatures() {
        let env = Env::default();
        let base_vaa = base_vaa_with_guardian_set(&env, 0);
        let (sig_entry, valid_eth_address) = sign_for_vaa(&env, &base_vaa, [0x55u8; 32], 0);

        let mut sigs = Vec::new(&env);
        sigs.push_back(sig_entry);

        let mut vaa = base_vaa.clone();
        vaa.signatures = sigs;

        let initial_guardians = soroban_sdk::vec![&env, valid_eth_address];
        let governance_emitter = BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(crate::Wormhole, (initial_guardians, governance_emitter));

        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Ok(true));
        });
    }

    #[test]
    fn test_verify_vaa_guardian_set_not_found() {
        let env = Env::default();
        let (contract_id, _client) = deploy_initialized(&env);
        let vaa = base_vaa_with_guardian_set(&env, 42);

        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Err(WormholeError::GuardianSetNotFound));
        });
    }

    #[test]
    fn test_verify_vaa_guardian_set_expired() {
        let env = Env::default();
        let (contract_id, _client) = deploy_initialized(&env);
        env.ledger().set_timestamp(1);
        env.as_contract(&contract_id, || {
            env.storage()
                .persistent()
                .set(&crate::storage::StorageKey::GuardianSetExpiry(0), &0u64);
        });

        let vaa = base_vaa_with_guardian_set(&env, 0);
        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Err(WormholeError::GuardianSetExpired));
        });
    }

    #[test]
    fn test_verify_vaa_guardian_index_out_of_bounds() {
        let env = Env::default();
        let (contract_id, _client) = deploy_initialized(&env);

        let mut vaa = base_vaa_with_guardian_set(&env, 0);
        let mut sigs = Vec::new(&env);
        sigs.push_back(Signature {
            guardian_index: 1,
            r: BytesN::<32>::from_array(&env, &[0u8; 32]),
            s: BytesN::<32>::from_array(&env, &[0u8; 32]),
            v: 0,
        });
        vaa.signatures = sigs;

        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Err(WormholeError::GuardianIndexOutOfBounds));
        });
    }

    #[test]
    fn test_verify_vaa_signatures_not_ascending() {
        let env = Env::default();
        let base_vaa = base_vaa_with_guardian_set(&env, 0);
        let (sig, guardian_eth) = sign_for_vaa(&env, &base_vaa, [0x77u8; 32], 1);
        let initial_guardians = soroban_sdk::vec![&env, guardian_eth.clone(), guardian_eth];
        let governance_emitter = BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(crate::Wormhole, (initial_guardians, governance_emitter));

        let mut vaa = base_vaa.clone();
        let mut sigs = Vec::new(&env);
        sigs.push_back(sig.clone());
        sigs.push_back(sig);
        vaa.signatures = sigs;

        env.as_contract(&contract_id, || {
            let res = verify_vaa_signatures(&vaa, &env);
            assert_eq!(res, Err(WormholeError::SignaturesNotAscending));
        });
    }
}
