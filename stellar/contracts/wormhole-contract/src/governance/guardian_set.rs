//! Guardian set upgrade governance action (`ACTION_GUARDIAN_SET_UPGRADE`, value
//! 2).
//!
//! Manages guardian set storage and upgrades. Guardian sets are indexed
//! sequentially starting from 0. When a new set is installed, the previous
//! set is marked with an expiry timestamp (24 hours) to allow in-flight
//! VAAs to still be processed.

use crate::{
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    storage::StorageKey,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, Vec, contractevent};
use wormhole_soroban_client::{
    ACTION_GUARDIAN_SET_UPGRADE, BytesReader, GUARDIAN_SET_EXPIRATION_TIME,
    GUARDIAN_SET_UPGRADE_PAYLOAD_MIN_LENGTH, GuardianSetInfo, STORAGE_TTL_EXTENSION,
    STORAGE_TTL_THRESHOLD, WormholeError,
};

/// Emitted when a guardian set upgrade is executed successfully.
///
/// Topics: `["wormhole_core", "guardian_set_upgrade"]`
#[contractevent(topics = ["wormhole_core", "guardian_set_upgrade"])]
struct GuardianSetUpgradeEvent {
    /// Index of the newly installed guardian set.
    new_guardian_set_index: u32,
    /// Number of guardians in the new set.
    guardian_count: u32,
}

/// Parsed payload for guardian set upgrade governance action.
#[derive(Debug, PartialEq)]
pub struct GuardianSetUpgradePayload {
    /// Module identifier (must be "Core").
    pub module: BytesN<32>,
    /// Governance action ID from the VAA header.
    ///
    /// Must equal `ACTION_GUARDIAN_SET_UPGRADE` (2); any other value is
    /// rejected by `validate_governance_header`.
    pub action: u8,
    /// Target chain (0 for all, 61 for Stellar).
    pub chain: u16,
    /// Index for the new guardian set (must be current + 1).
    pub new_guardian_set_index: u32,
    /// Ethereum addresses (20 bytes) of guardians in the new set.
    pub new_guardian_set_keys: Vec<BytesN<20>>,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for GuardianSetUpgradePayload {
    type Error = WormholeError;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < GUARDIAN_SET_UPGRADE_PAYLOAD_MIN_LENGTH {
            return Err(WormholeError::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;

        let new_guardian_set_index = reader.read_u32_be()?;
        let guardian_count = reader.read_u8()?;

        let mut keys = Vec::new(env);
        for _ in 0..guardian_count {
            let key = reader.read_bytes_n::<20>()?;
            keys.push_back(BytesN::from_array(env, &key.to_array()));
        }

        if reader.remaining() != 0 {
            return Err(WormholeError::InvalidPayload);
        }

        Ok(GuardianSetUpgradePayload {
            module,
            action,
            chain,
            new_guardian_set_index,
            new_guardian_set_keys: keys,
        })
    }
}

impl GuardianSetUpgradePayload {
    fn validate(&self, env: &Env) -> Result<(), WormholeError> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_GUARDIAN_SET_UPGRADE,
        )?;

        let current_index = get_current_guardian_set_index(env);
        if self.new_guardian_set_index != current_index.saturating_add(1) {
            return Err(WormholeError::InvalidGuardianSetSequence);
        }

        if self.new_guardian_set_keys.is_empty() {
            return Err(WormholeError::EmptyGuardianSet);
        }

        Ok(())
    }
}

/// Returns the current active guardian set index.
pub fn get_current_guardian_set_index(env: &Env) -> u32 {
    env.storage()
        .persistent()
        .get(&StorageKey::CurrentGuardianSetIndex)
        .unwrap_or(0)
}

/// Updates the current guardian set index pointer.
pub fn set_current_index(env: &Env, index: u32) {
    env.storage()
        .persistent()
        .set(&StorageKey::CurrentGuardianSetIndex, &index);

    env.storage().persistent().extend_ttl(
        &StorageKey::CurrentGuardianSetIndex,
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

/// Retrieves a guardian set by index.
pub fn get_guardian_set(env: &Env, index: u32) -> Result<GuardianSetInfo, WormholeError> {
    env.storage()
        .persistent()
        .get(&StorageKey::GuardianSet(index))
        .ok_or(WormholeError::GuardianSetNotFound)
}

/// Returns the expiry timestamp for a guardian set, if set.
pub fn get_guardian_set_expiry(env: &Env, index: u32) -> Option<u64> {
    env.storage()
        .persistent()
        .get(&StorageKey::GuardianSetExpiry(index))
}

/// Stores a new guardian set at the given index.
///
/// Index 0 is only valid during constructor (guardian set won't exist yet).
/// Subsequent indices must be strictly sequential (current + 1).
/// Cannot overwrite existing guardian sets.
pub fn store(env: &Env, index: u32, set: GuardianSetInfo) -> Result<(), WormholeError> {
    let current_index = get_current_guardian_set_index(env);

    // Index 0 is only valid during constructor (guardian set won't exist yet)
    // All other indices must be current + 1 for proper sequencing
    if index != 0 && index != current_index.saturating_add(1) {
        return Err(WormholeError::InvalidGuardianSetSequence);
    }

    // Cannot overwrite existing guardian sets (including index 0 after constructor)
    if get_guardian_set(env, index).is_ok() {
        return Err(WormholeError::GuardianSetAlreadyExists);
    }

    env.storage()
        .persistent()
        .set(&StorageKey::GuardianSet(index), &set);

    env.storage().persistent().extend_ttl(
        &StorageKey::GuardianSet(index),
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );

    Ok(())
}

/// Sets the expiry timestamp for a guardian set.
fn set_expiry(env: &Env, index: u32, expiry: u64) {
    env.storage()
        .persistent()
        .set(&StorageKey::GuardianSetExpiry(index), &expiry);

    env.storage().persistent().extend_ttl(
        &StorageKey::GuardianSetExpiry(index),
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

/// Governance action handler for guardian set upgrades.
pub struct GuardianSetUpgradeAction;

impl GovernanceAction for GuardianSetUpgradeAction {
    type Payload = GuardianSetUpgradePayload;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), WormholeError> {
        payload.validate(env)
    }

    fn execute(
        env: &Env,
        vaa: &crate::vaa::VAA,
        payload: &Self::Payload,
    ) -> Result<(), WormholeError> {
        let old_index = get_current_guardian_set_index(env);

        let expiry_time =
            u64::from(vaa.timestamp).saturating_add(u64::from(GUARDIAN_SET_EXPIRATION_TIME));
        set_expiry(env, old_index, expiry_time);

        let new_set = GuardianSetInfo {
            keys: payload.new_guardian_set_keys.clone(),
            creation_time: env.ledger().timestamp(),
        };
        store(env, payload.new_guardian_set_index, new_set)?;

        set_current_index(env, payload.new_guardian_set_index);

        GuardianSetUpgradeEvent {
            new_guardian_set_index: payload.new_guardian_set_index,
            guardian_count: payload.new_guardian_set_keys.len(),
        }
        .publish(env);

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Wormhole;
    use soroban_sdk::{Bytes, BytesN, Env, IntoVal, Symbol, Val, testutils::Events, vec};
    use wormhole_soroban_client::{
        CHAIN_ID_STELLAR, ConsistencyLevel, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, MODULE_CORE,
        Signature, VAA,
    };

    fn deploy_initialized(env: &Env) -> soroban_sdk::Address {
        let guardian = BytesN::<20>::from_array(env, &[0u8; 20]);
        let initial_guardians = vec![env, guardian];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        env.register(Wormhole, (initial_guardians, governance_emitter))
    }

    fn build_payload(
        env: &Env,
        module: [u8; 32],
        action: u8,
        chain: u16,
        new_index: u32,
        keys: &[BytesN<20>],
    ) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &module));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_slice(env, &new_index.to_be_bytes()));
        payload.push_back(keys.len() as u8);
        for k in keys {
            payload.append(&Bytes::from_array(env, &k.to_array()));
        }
        payload
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

    #[test]
    fn test_guardian_set_upgrade_payload_rejects_short_payload() {
        let env = Env::default();
        let short = Bytes::from_slice(&env, &[0u8; 10]);
        let res = GuardianSetUpgradePayload::try_from((&env, &short));
        assert_eq!(res, Err(WormholeError::InvalidPayload));
    }

    #[test]
    fn test_guardian_set_upgrade_validate_rejects_non_sequential_index() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        let payload = GuardianSetUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_GUARDIAN_SET_UPGRADE,
            chain: CHAIN_ID_STELLAR,
            new_guardian_set_index: 2,
            new_guardian_set_keys: vec![&env, BytesN::from_array(&env, &[9u8; 20])],
        };

        env.as_contract(&contract_id, || {
            let res = payload.validate(&env);
            assert_eq!(res, Err(WormholeError::InvalidGuardianSetSequence));
        });
    }

    #[test]
    fn test_guardian_set_upgrade_validate_rejects_empty_keys() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        let payload = GuardianSetUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_GUARDIAN_SET_UPGRADE,
            chain: CHAIN_ID_STELLAR,
            new_guardian_set_index: 1,
            new_guardian_set_keys: Vec::new(&env),
        };

        env.as_contract(&contract_id, || {
            let res = payload.validate(&env);
            assert_eq!(res, Err(WormholeError::EmptyGuardianSet));
        });
    }

    #[test]
    fn test_guardian_set_upgrade_execute_updates_state() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        let new_guardian = BytesN::<20>::from_array(&env, &[7u8; 20]);
        let bytes = build_payload(
            &env,
            MODULE_CORE,
            ACTION_GUARDIAN_SET_UPGRADE,
            CHAIN_ID_STELLAR,
            1,
            core::slice::from_ref(&new_guardian),
        );
        let payload = GuardianSetUpgradePayload::try_from((&env, &bytes)).unwrap();
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![&env],
            timestamp: 100,
            nonce: 1,
            emitter_chain: GOVERNANCE_CHAIN_ID,
            emitter_address: BytesN::from_array(&env, &[0u8; 32]),
            sequence: 1,
            consistency_level: wormhole_soroban_client::ConsistencyLevel::Confirmed,
            payload: Bytes::new(&env),
        };

        env.as_contract(&contract_id, || {
            GuardianSetUpgradeAction::execute(&env, &vaa, &payload).unwrap();

            assert_eq!(get_current_guardian_set_index(&env), 1);
            let g1 = get_guardian_set(&env, 1).unwrap();
            assert_eq!(g1.keys.len(), 1);
            assert_eq!(g1.keys.get(0).unwrap(), new_guardian);

            let expiry = get_guardian_set_expiry(&env, 0).unwrap();
            assert_eq!(
                expiry,
                u64::from(vaa.timestamp) + u64::from(GUARDIAN_SET_EXPIRATION_TIME)
            );
        });

        let events = env.events().all().filter_by_contract(&contract_id);
        let guardian_count_val: Val = 1u32.into_val(&env);
        let new_index_val: Val = 1u32.into_val(&env);
        assert_eq!(
            events,
            vec![
                &env,
                (
                    contract_id,
                    vec![
                        &env,
                        Symbol::new(&env, "wormhole_core").into_val(&env),
                        Symbol::new(&env, "guardian_set_upgrade").into_val(&env),
                    ],
                    soroban_sdk::map![
                        &env,
                        (Symbol::new(&env, "guardian_count"), guardian_count_val),
                        (Symbol::new(&env, "new_guardian_set_index"), new_index_val),
                    ]
                    .into_val(&env),
                )
            ]
        );
    }

    #[test]
    fn test_store_rejects_overwrite() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        env.as_contract(&contract_id, || {
            let set = GuardianSetInfo {
                keys: vec![&env, BytesN::from_array(&env, &[1u8; 20])],
                creation_time: 0,
            };
            let res = store(&env, 0, set);
            assert_eq!(res, Err(WormholeError::GuardianSetAlreadyExists));
        });
    }

    #[test]
    fn test_submit_rejects_malicious_old_guardian_index() {
        let env = Env::default();

        let payload = build_payload(
            &env,
            MODULE_CORE,
            ACTION_GUARDIAN_SET_UPGRADE,
            CHAIN_ID_STELLAR,
            0,
            &[BytesN::<20>::from_array(&env, &[5u8; 20])],
        );
        let base = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: Vec::new(&env),
            timestamp: 1,
            nonce: 1,
            emitter_chain: GOVERNANCE_CHAIN_ID,
            emitter_address: BytesN::from_array(&env, &GOVERNANCE_EMITTER),
            sequence: 1,
            consistency_level: ConsistencyLevel::Confirmed,
            payload,
        };
        let (signed, guardian_eth) = sign_vaa_with_key(&env, base, [0x44u8; 32]);
        let contract_id = env.register(
            Wormhole,
            (
                vec![&env, guardian_eth],
                BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER),
            ),
        );
        let vaa_bytes = serialize_vaa(&env, &signed);

        env.as_contract(&contract_id, || {
            let res = GuardianSetUpgradeAction::submit(&env, vaa_bytes.clone());
            assert_eq!(res, Err(WormholeError::InvalidGuardianSetSequence));
        });
    }

    #[test]
    fn test_guardian_set_expiry_arithmetic_bounds() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        let new_guardian = BytesN::<20>::from_array(&env, &[9u8; 20]);
        let payload = GuardianSetUpgradePayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_GUARDIAN_SET_UPGRADE,
            chain: CHAIN_ID_STELLAR,
            new_guardian_set_index: 1,
            new_guardian_set_keys: vec![&env, new_guardian],
        };
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![&env],
            timestamp: u32::MAX,
            nonce: 0,
            emitter_chain: GOVERNANCE_CHAIN_ID,
            emitter_address: BytesN::from_array(&env, &[0u8; 32]),
            sequence: 0,
            consistency_level: ConsistencyLevel::Confirmed,
            payload: Bytes::new(&env),
        };
        env.as_contract(&contract_id, || {
            GuardianSetUpgradeAction::execute(&env, &vaa, &payload).unwrap();
            let expiry = get_guardian_set_expiry(&env, 0).unwrap();
            assert_eq!(
                expiry,
                u64::from(u32::MAX).saturating_add(u64::from(GUARDIAN_SET_EXPIRATION_TIME))
            );
        });
    }
}
