//! Common governance action trait and processing utilities.
//!
//! Defines the [`GovernanceAction`] trait that all governance actions
//! implement, providing a standard flow: verify VAA → check replay → parse
//! payload → validate → consume → execute.

use crate::{storage::StorageKey, utils::keccak256_hash, vaa::verify_vaa_signatures};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env};
use wormhole_soroban_client::{
    BytesReader, CHAIN_ID_STELLAR, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, MODULE_CORE, VAA,
    WormholeError,
};

/// Trait for implementing Wormhole governance actions.
///
/// Each action defines a `Payload` type with parsing via `TryFrom`, validation
/// logic, and execution behavior. The `submit` method provides the standard
/// governance flow with replay protection.
///
/// **Critical**: VAA consumption happens BEFORE execution to prevent replay
/// if the execution modifies contract behavior (e.g., contract upgrades).
pub trait GovernanceAction {
    /// Payload type for this action, parsed from VAA payload bytes.
    type Payload: for<'a> TryFrom<(&'a Env, &'a Bytes), Error = WormholeError>;

    /// Validates the parsed payload (e.g., sequential guardian set index).
    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), WormholeError>;

    /// Executes the governance action after all validation passes.
    fn execute(env: &Env, vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError>;

    /// Standard governance submission flow with replay protection.
    ///
    /// 1. Parse and verify VAA (signatures + governance source)
    /// 2. Check VAA not already consumed
    /// 3. Parse and validate payload
    /// 4. **Consume VAA (before execution!)**
    /// 5. Execute the action
    fn submit(env: &Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        let (vaa, vaa_hash) = verify_and_hash_governance_vaa(env, &vaa_bytes)?;
        require_vaa_not_consumed(env, &vaa_hash)?;

        let payload = Self::Payload::try_from((env, &vaa.payload))?;
        Self::validate_payload(env, &payload)?;

        consume_vaa(env, &vaa_hash);

        Self::execute(env, &vaa, &payload)?;

        Ok(())
    }
}

/// Validates that a VAA originates from the authorized governance source.
fn verify_governance_vaa(vaa: &VAA) -> Result<(), WormholeError> {
    if vaa.emitter_chain != GOVERNANCE_CHAIN_ID {
        return Err(WormholeError::InvalidGovernanceChain);
    }

    if vaa.emitter_address.to_array() != GOVERNANCE_EMITTER {
        return Err(WormholeError::InvalidGovernanceEmitter);
    }

    Ok(())
}

/// Parses, verifies, and hashes a governance VAA for consumption tracking.
fn verify_and_hash_governance_vaa(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<(VAA, BytesN<32>), WormholeError> {
    let vaa = VAA::try_from((env, vaa_bytes))?;
    verify_governance_vaa(&vaa)?;
    verify_vaa_signatures(&vaa, env)?;

    let body_bytes = vaa.serialize_body(env);
    let vaa_hash = keccak256_hash(env, &body_bytes);

    Ok((vaa, vaa_hash))
}

/// Checks if a VAA body hash has been consumed.
fn is_vaa_consumed(env: &Env, hash: &BytesN<32>) -> bool {
    env.storage()
        .persistent()
        .has(&StorageKey::ConsumedGovernanceVAA(hash.clone()))
}

/// Returns error if VAA has already been consumed (replay protection).
fn require_vaa_not_consumed(env: &Env, hash: &BytesN<32>) -> Result<(), WormholeError> {
    if is_vaa_consumed(env, hash) {
        return Err(WormholeError::GovernanceVAAAlreadyConsumed);
    }
    Ok(())
}

/// Marks a VAA as consumed to prevent replay attacks.
///
/// Uses persistent storage, which is archived (not deleted) on TTL expiry.
/// An attacker replaying a VAA whose entry was archived must first restore
/// it via `RestoreFootprint`, at which point the consumed flag blocks replay.
fn consume_vaa(env: &Env, hash: &BytesN<32>) {
    env.storage()
        .persistent()
        .set(&StorageKey::ConsumedGovernanceVAA(hash.clone()), &true);
}

/// Checks if a governance VAA has been consumed without full verification.
pub fn is_governance_vaa_consumed_from_bytes(
    env: &Env,
    vaa_bytes: &Bytes,
) -> Result<bool, WormholeError> {
    let body_bytes = VAA::get_body_bytes(vaa_bytes)?;
    let vaa_hash = keccak256_hash(env, &body_bytes);
    Ok(is_vaa_consumed(env, &vaa_hash))
}

/// Parses the 35-byte governance header: module (32) + action (1) + chain (2).
pub fn parse_governance_header(
    env: &Env,
    reader: &mut BytesReader,
) -> Result<(BytesN<32>, u8, u16), WormholeError> {
    let module = reader.read_bytes_n::<32>()?;
    let action = reader.read_u8()?;
    let chain = reader.read_u16_be()?;

    Ok((BytesN::from_array(env, &module.to_array()), action, chain))
}

/// Validates governance header fields against expected values.
///
/// - Module must be "Core" (right-padded)
/// - Action must match the expected action ID
/// - Chain must be 0 (all chains) or 61 (Stellar)
pub fn validate_governance_header(
    module: &BytesN<32>,
    action: u8,
    chain: u16,
    expected_action: u8,
) -> Result<(), WormholeError> {
    if module.to_array() != MODULE_CORE {
        return Err(WormholeError::InvalidGovernanceModule);
    }

    if action != expected_action {
        return Err(WormholeError::InvalidGovernanceAction);
    }

    if chain != 0 && chain != CHAIN_ID_STELLAR {
        return Err(WormholeError::InvalidGovernanceChain);
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        Wormhole,
        governance::{
            GuardianSetUpgradeAction, SetMessageFeeAction, get_message_fee,
            guardian_set::get_current_guardian_set_index,
        },
    };
    use soroban_sdk::{
        Bytes, BytesN, Env, IntoVal, Symbol, Vec,
        testutils::{Events, Ledger},
        vec,
    };
    use wormhole_soroban_client::{
        ACTION_GUARDIAN_SET_UPGRADE, ACTION_SET_MESSAGE_FEE, ConsistencyLevel, GOVERNANCE_CHAIN_ID,
        GOVERNANCE_EMITTER, MODULE_CORE, SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH, Signature, VAA,
    };

    fn deploy_with_guardian(env: &Env, guardian_eth: BytesN<20>) -> soroban_sdk::Address {
        let initial_guardians = vec![env, guardian_eth];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        env.register(Wormhole, (initial_guardians, governance_emitter))
    }

    fn serialize_vaa(env: &Env, vaa: &VAA) -> Bytes {
        let mut out = Bytes::new(env);
        out.push_back(vaa.version as u8);
        out.append(&Bytes::from_slice(
            env,
            &vaa.guardian_set_index.to_be_bytes(),
        ));
        out.push_back(vaa.signatures.len() as u8);

        for sig in vaa.signatures.iter() {
            out.push_back(sig.guardian_index as u8);
            out.append(&Bytes::from_array(env, &sig.r.to_array()));
            out.append(&Bytes::from_array(env, &sig.s.to_array()));
            out.push_back(sig.v as u8);
        }

        out.append(&Bytes::from_slice(env, &vaa.timestamp.to_be_bytes()));
        out.append(&Bytes::from_slice(env, &vaa.nonce.to_be_bytes()));
        out.append(&Bytes::from_slice(
            env,
            &(vaa.emitter_chain as u16).to_be_bytes(),
        ));
        out.append(&Bytes::from_array(env, &vaa.emitter_address.to_array()));
        out.append(&Bytes::from_slice(env, &vaa.sequence.to_be_bytes()));
        out.push_back(vaa.consistency_level as u8);
        out.append(&vaa.payload);
        out
    }

    fn build_set_fee_payload(env: &Env, action: u8, chain: u16, fee: u64) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &MODULE_CORE));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[0u8; 24]));
        payload.append(&Bytes::from_slice(env, &fee.to_be_bytes()));
        payload
    }

    fn build_guardian_upgrade_payload(
        env: &Env,
        action: u8,
        chain: u16,
        new_index: u32,
        key: BytesN<20>,
    ) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &MODULE_CORE));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_slice(env, &new_index.to_be_bytes()));
        payload.push_back(1);
        payload.append(&Bytes::from_array(env, &key.to_array()));
        payload
    }

    fn sign_vaa_with_key(env: &Env, mut vaa: VAA, sk_bytes: [u8; 32]) -> (VAA, BytesN<20>) {
        let body = vaa.serialize_body(env);
        let body_hash: Bytes = crate::utils::keccak256_hash(env, &body).into();
        let double_hash = env.crypto().keccak256(&body_hash).to_array();

        let secp = secp256k1::Secp256k1::new();
        let sk = secp256k1::SecretKey::from_byte_array(sk_bytes).unwrap();
        let pk = secp256k1::PublicKey::from_secret_key(&secp, &sk);
        let pk_bytes65 = BytesN::<65>::from_array(env, &pk.serialize_uncompressed());
        let guardian_eth = crate::utils::pubkey_to_eth_address(env, &pk_bytes65);

        let message = secp256k1::Message::from_digest(double_hash);
        let sig = secp.sign_ecdsa_recoverable(message, &sk);
        let (recid, compact64) = sig.serialize_compact();

        let mut r = [0u8; 32];
        let mut s = [0u8; 32];
        r.copy_from_slice(&compact64[0..32]);
        s.copy_from_slice(&compact64[32..64]);

        let mut sigs = Vec::new(env);
        sigs.push_back(Signature {
            guardian_index: 0,
            r: BytesN::from_array(env, &r),
            s: BytesN::from_array(env, &s),
            v: i32::from(recid) as u32,
        });
        vaa.signatures = sigs;
        (vaa, guardian_eth)
    }

    fn build_signed_set_fee_vaa(
        env: &Env,
        emitter_chain: u32,
        emitter_address: [u8; 32],
        fee: u64,
    ) -> (Bytes, BytesN<20>) {
        let payload = build_set_fee_payload(env, ACTION_SET_MESSAGE_FEE, 61, fee);
        let base = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: Vec::new(env),
            timestamp: 1,
            nonce: 7,
            emitter_chain,
            emitter_address: BytesN::<32>::from_array(env, &emitter_address),
            sequence: 42,
            consistency_level: ConsistencyLevel::Confirmed,
            payload,
        };
        let (signed, guardian_eth) = sign_vaa_with_key(env, base, [0x33u8; 32]);
        (serialize_vaa(env, &signed), guardian_eth)
    }

    fn build_signed_guardian_upgrade_vaa(
        env: &Env,
        emitter_chain: u32,
        emitter_address: [u8; 32],
        new_index: u32,
        new_key: BytesN<20>,
    ) -> (Bytes, BytesN<20>) {
        let payload = build_guardian_upgrade_payload(
            env,
            ACTION_GUARDIAN_SET_UPGRADE,
            61,
            new_index,
            new_key,
        );
        let base = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: Vec::new(env),
            timestamp: 2,
            nonce: 8,
            emitter_chain,
            emitter_address: BytesN::<32>::from_array(env, &emitter_address),
            sequence: 43,
            consistency_level: ConsistencyLevel::Confirmed,
            payload,
        };
        let (signed, guardian_eth) = sign_vaa_with_key(env, base, [0x33u8; 32]);
        (serialize_vaa(env, &signed), guardian_eth)
    }

    #[test]
    fn test_validate_governance_header_errors() {
        let bad_module = BytesN::<32>::from_array(&Env::default(), &[1u8; 32]);
        assert_eq!(
            validate_governance_header(
                &bad_module,
                ACTION_SET_MESSAGE_FEE,
                61,
                ACTION_SET_MESSAGE_FEE
            ),
            Err(WormholeError::InvalidGovernanceModule)
        );

        let good_module = BytesN::<32>::from_array(&Env::default(), &MODULE_CORE);
        assert_eq!(
            validate_governance_header(&good_module, 9, 61, ACTION_SET_MESSAGE_FEE),
            Err(WormholeError::InvalidGovernanceAction)
        );
        assert_eq!(
            validate_governance_header(
                &good_module,
                ACTION_SET_MESSAGE_FEE,
                2,
                ACTION_SET_MESSAGE_FEE
            ),
            Err(WormholeError::InvalidGovernanceChain)
        );
        assert!(
            validate_governance_header(
                &good_module,
                ACTION_SET_MESSAGE_FEE,
                0,
                ACTION_SET_MESSAGE_FEE
            )
            .is_ok()
        );
        assert!(
            validate_governance_header(
                &good_module,
                ACTION_SET_MESSAGE_FEE,
                61,
                ACTION_SET_MESSAGE_FEE
            )
            .is_ok()
        );
    }

    #[test]
    fn test_is_governance_vaa_consumed_invalid_bytes() {
        let env = Env::default();
        let malformed = Bytes::new(&env);
        let res = is_governance_vaa_consumed_from_bytes(&env, &malformed);
        assert_eq!(res, Err(WormholeError::InvalidVAAFormat));
    }

    #[test]
    fn test_is_governance_vaa_consumed_roundtrip() {
        let env = Env::default();
        let contract_id = env.register(
            Wormhole,
            (
                vec![&env, BytesN::<20>::from_array(&env, &[0u8; 20])],
                BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER),
            ),
        );
        let body = Bytes::from_slice(&env, &[1u8; 51]);
        let mut vaa_bytes = Bytes::new(&env);
        vaa_bytes.append(&Bytes::from_slice(&env, &[1u8]));
        vaa_bytes.append(&Bytes::from_slice(&env, &0u32.to_be_bytes()));
        vaa_bytes.append(&Bytes::from_slice(&env, &[0u8]));
        vaa_bytes.append(&body);

        env.as_contract(&contract_id, || {
            let hash = keccak256_hash(&env, &body);
            env.storage()
                .persistent()
                .set(&StorageKey::ConsumedGovernanceVAA(hash), &true);

            let consumed = is_governance_vaa_consumed_from_bytes(&env, &vaa_bytes).unwrap();
            assert!(consumed);
        });
    }

    #[test]
    fn test_submit_replay_and_source_checks() {
        let env = Env::default();
        env.ledger().set_timestamp(10);

        let (vaa_bytes, guardian_eth) =
            build_signed_set_fee_vaa(&env, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, 55);
        let contract_id = deploy_with_guardian(&env, guardian_eth);

        env.as_contract(&contract_id, || {
            let first = SetMessageFeeAction::submit(&env, vaa_bytes.clone());
            assert_eq!(first, Ok(()));
            assert_eq!(get_message_fee(&env), 55);
            let second = SetMessageFeeAction::submit(&env, vaa_bytes.clone());
            assert_eq!(second, Err(WormholeError::GovernanceVAAAlreadyConsumed));
        });
        let events = env.events().all().filter_by_contract(&contract_id);
        assert_eq!(
            events,
            vec![
                &env,
                (
                    contract_id.clone(),
                    vec![
                        &env,
                        Symbol::new(&env, "wormhole_core").into_val(&env),
                        Symbol::new(&env, "fee_set").into_val(&env),
                    ],
                    55u64.into_val(&env),
                )
            ]
        );

        let (wrong_chain_vaa, guardian_eth_2) =
            build_signed_set_fee_vaa(&env, 9, GOVERNANCE_EMITTER, 77);
        let contract_id_2 = deploy_with_guardian(&env, guardian_eth_2);
        env.as_contract(&contract_id_2, || {
            let res = SetMessageFeeAction::submit(&env, wrong_chain_vaa.clone());
            assert_eq!(res, Err(WormholeError::InvalidGovernanceChain));
        });

        let mut wrong_emitter = GOVERNANCE_EMITTER;
        wrong_emitter[31] = 7;
        let (wrong_emitter_vaa, guardian_eth_3) =
            build_signed_set_fee_vaa(&env, GOVERNANCE_CHAIN_ID, wrong_emitter, 88);
        let contract_id_3 = deploy_with_guardian(&env, guardian_eth_3);
        env.as_contract(&contract_id_3, || {
            let res = SetMessageFeeAction::submit(&env, wrong_emitter_vaa.clone());
            assert_eq!(res, Err(WormholeError::InvalidGovernanceEmitter));
        });
    }

    #[test]
    fn test_cross_action_replay_domains_do_not_collide() {
        let env = Env::default();
        env.ledger().set_timestamp(20);

        let (vaa_fee, guardian_eth) =
            build_signed_set_fee_vaa(&env, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, 99);
        let new_guardian = BytesN::<20>::from_array(&env, &[9u8; 20]);
        let (vaa_guardian, _guardian_eth_2) = build_signed_guardian_upgrade_vaa(
            &env,
            GOVERNANCE_CHAIN_ID,
            GOVERNANCE_EMITTER,
            1,
            new_guardian,
        );

        let contract_id = deploy_with_guardian(&env, guardian_eth);
        env.as_contract(&contract_id, || {
            assert_eq!(SetMessageFeeAction::submit(&env, vaa_fee.clone()), Ok(()));
            assert_eq!(
                GuardianSetUpgradeAction::submit(&env, vaa_guardian.clone()),
                Ok(())
            );
            assert_eq!(get_message_fee(&env), 99);
            assert_eq!(get_current_guardian_set_index(&env), 1);

            assert_eq!(
                SetMessageFeeAction::submit(&env, vaa_fee.clone()),
                Err(WormholeError::GovernanceVAAAlreadyConsumed)
            );
            assert_eq!(
                GuardianSetUpgradeAction::submit(&env, vaa_guardian.clone()),
                Err(WormholeError::GovernanceVAAAlreadyConsumed)
            );
        });
    }

    #[test]
    fn test_set_fee_payload_length_sanity() {
        let env = Env::default();
        let payload = build_set_fee_payload(&env, ACTION_SET_MESSAGE_FEE, 61, 1);
        assert_eq!(payload.len(), SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH);
    }
}
