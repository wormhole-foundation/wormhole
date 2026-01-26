//! Guardian set upgrade governance action (`ACTION_GUARDIAN_SET_UPGRADE`, value 2).
//!
//! Manages guardian set storage and upgrades. Guardian sets are indexed
//! sequentially starting from 0. When a new set is installed, the previous
//! set is marked with an expiry timestamp (24 hours) to allow in-flight
//! VAAs to still be processed.

use crate::{
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    initialize,
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
    /// Must equal `ACTION_GUARDIAN_SET_UPGRADE` (2); any other value is rejected by
    /// `validate_governance_header`.
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
/// Index 0 is only valid during initialization. Subsequent indices must be
/// strictly sequential (current + 1) and cannot overwrite existing sets.
pub fn store(env: &Env, index: u32, set: GuardianSetInfo) -> Result<(), WormholeError> {
    let current_index = get_current_guardian_set_index(env);

    if index == 0 {
        if initialize::is_initialized(env) {
            return Err(WormholeError::GuardianSetAlreadyExists);
        }
    } else if index != current_index.saturating_add(1) {
        return Err(WormholeError::InvalidGuardianSetSequence);
    }

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

/// Marks a guardian set to expire in 24 hours from now.
fn expire_guardian_set(env: &Env, index: u32) {
    let expiry = env
        .ledger()
        .timestamp()
        .saturating_add(u64::from(GUARDIAN_SET_EXPIRATION_TIME));
    set_expiry(env, index, expiry);
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
