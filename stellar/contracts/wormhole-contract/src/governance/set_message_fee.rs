//! Set message fee governance action (`ACTION_SET_MESSAGE_FEE`, value 3).
//!
//! Allows guardians to update the fee charged for posting cross-chain messages.
//! The fee is denominated in stroops (10^-7 XLM).

use crate::{
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    storage::StorageKey,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, contractevent};
use wormhole_soroban_client::{
    ACTION_SET_MESSAGE_FEE, BytesReader, SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH, STORAGE_TTL_EXTENSION,
    STORAGE_TTL_THRESHOLD, U256_PADDING_BYTES, VAA, WormholeError,
};

/// Emitted when the message fee is updated.
#[contractevent(topics = ["wormhole_core", "fee_set"], data_format = "single-value")]
pub struct MessageFeeSetEvent {
    /// New fee in stroops.
    pub fee: u64,
}

/// Parsed payload for set message fee governance action.
#[derive(Debug, PartialEq)]
pub struct SetMessageFeePayload {
    /// Module identifier (must be "Core").
    pub module: BytesN<32>,
    /// Governance action ID from the VAA header.
    ///
    /// Must equal `ACTION_SET_MESSAGE_FEE` (3); any other value is rejected by
    /// `validate_governance_header`.
    pub action: u8,
    /// Target chain (0 for all, 61 for Stellar).
    pub chain: u16,
    /// New message fee in stroops.
    pub fee: u64,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for SetMessageFeePayload {
    type Error = WormholeError;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH {
            return Err(WormholeError::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;

        reader.skip(U256_PADDING_BYTES)?;
        let fee = reader.read_u64_be()?;

        Ok(SetMessageFeePayload {
            module,
            action,
            chain,
            fee,
        })
    }
}

impl SetMessageFeePayload {
    fn validate(&self) -> Result<(), WormholeError> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_SET_MESSAGE_FEE,
        )
    }
}

/// Returns the current message fee in stroops, or 0 if not set.
pub fn get_message_fee(env: &Env) -> u64 {
    env.storage()
        .persistent()
        .get(&StorageKey::MessageFee)
        .unwrap_or(0)
}

fn set_message_fee(env: &Env, fee: u64) {
    env.storage()
        .persistent()
        .set(&StorageKey::MessageFee, &fee);

    env.storage().persistent().extend_ttl(
        &StorageKey::MessageFee,
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

/// Governance action handler for setting the message fee.
pub struct SetMessageFeeAction;

impl GovernanceAction for SetMessageFeeAction {
    type Payload = SetMessageFeePayload;

    fn validate_payload(_env: &Env, payload: &Self::Payload) -> Result<(), WormholeError> {
        payload.validate()
    }

    fn execute(env: &Env, _vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError> {
        set_message_fee(env, payload.fee);
        MessageFeeSetEvent { fee: payload.fee }.publish(env);
        Ok(())
    }
}
