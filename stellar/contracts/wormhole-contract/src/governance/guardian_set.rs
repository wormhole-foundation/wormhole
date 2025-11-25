use crate::{
    constants::*,
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    initialize,
    storage::StorageKey,
    utils::BytesReader,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, Vec, contractevent};
use wormhole_interface::{Error, GuardianSetInfo};

/// Event published when a guardian set upgrade is executed.
///
/// Topics: ["wormhole_core", "gs_upg"]
/// Data: { new_guardian_set_index: u32, guardian_count: u32 }
#[contractevent(topics = ["wormhole_core", "gs_upg"])]
struct GuardianSetUpgradeEvent {
    new_guardian_set_index: u32,
    guardian_count: u32,
}

#[derive(Debug, PartialEq)]
pub struct GuardianSetUpgradePayload {
    pub module: BytesN<32>,
    pub action: u8,
    pub chain: u16,
    pub new_guardian_set_index: u32,
    pub new_guardian_set_keys: Vec<BytesN<20>>,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for GuardianSetUpgradePayload {
    type Error = Error;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < GUARDIAN_SET_UPGRADE_PAYLOAD_MIN_LENGTH {
            return Err(Error::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;

        let new_guardian_set_index = reader.read_u32_be()?;
        let guardian_count = reader.read_u8()?;

        let mut keys = Vec::new(env);
        for _ in 0..guardian_count {
            let key = reader.read_bytes_n::<20>(env)?;
            keys.push_back(key);
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
    fn validate(&self, env: &Env) -> Result<(), Error> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_GUARDIAN_SET_UPGRADE,
        )?;

        let current_index = get_current_index(env);
        if self.new_guardian_set_index != current_index.saturating_add(1) {
            return Err(Error::InvalidGuardianSetSequence);
        }

        if self.new_guardian_set_keys.is_empty() {
            return Err(Error::EmptyGuardianSet);
        }

        Ok(())
    }
}

pub(crate) fn get_current_index(env: &Env) -> u32 {
    env.storage()
        .persistent()
        .get(&StorageKey::CurrentGuardianSetIndex)
        .unwrap_or(0)
}

pub(crate) fn set_current_index(env: &Env, index: u32) {
    env.storage()
        .persistent()
        .set(&StorageKey::CurrentGuardianSetIndex, &index);
}

pub(crate) fn get(env: &Env, index: u32) -> Result<GuardianSetInfo, Error> {
    env.storage()
        .persistent()
        .get(&StorageKey::GuardianSet(index))
        .ok_or(Error::GuardianSetNotFound)
}

pub(crate) fn store(env: &Env, index: u32, set: GuardianSetInfo) -> Result<(), Error> {
    let current_index = get_current_index(env);

    if index == 0 {
        if initialize::is_initialized(env) {
            return Err(Error::GuardianSetAlreadyExists);
        }
    } else if index != current_index.saturating_add(1) {
        return Err(Error::InvalidGuardianSetSequence);
    }

    if get(env, index).is_ok() {
        return Err(Error::GuardianSetAlreadyExists);
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

pub(crate) fn get_expiry(env: &Env, index: u32) -> Option<u64> {
    env.storage()
        .persistent()
        .get(&StorageKey::GuardianSetExpiry(index))
}

pub(crate) fn set_expiry(env: &Env, index: u32, expiry: u64) {
    env.storage()
        .persistent()
        .set(&StorageKey::GuardianSetExpiry(index), &expiry);
}

pub(crate) struct GuardianSetUpgradeAction;

impl GovernanceAction for GuardianSetUpgradeAction {
    type Payload = GuardianSetUpgradePayload;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), Error> {
        payload.validate(env)
    }

    fn execute(env: &Env, vaa: &crate::vaa::VAA, payload: &Self::Payload) -> Result<(), Error> {
        let old_index = get_current_index(env);

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
