use crate::{governance::action::{GovernanceAction, parse_governance_header, validate_governance_header}, storage::StorageKey};
use core::convert::TryFrom;
use soroban_sdk::{contractevent, Bytes, BytesN, Env};
use wormhole_soroban_client::{
    BytesReader, WormholeError, ACTION_SET_MESSAGE_FEE, SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH,
    STORAGE_TTL_EXTENSION, STORAGE_TTL_THRESHOLD, U256_PADDING_BYTES, VAA,
};

/// Event published when the message fee is updated.
///
/// Topics: ["wormhole_core", "fee_set"]
/// - "wormhole_core": Namespace for all core contract governance/lifecycle events
/// - "fee_set": Event type for message fee updates
/// Data: fee (u64) - uses single-value format for cleaner output
#[contractevent(topics = ["wormhole_core", "fee_set"], data_format = "single-value")]
pub struct MessageFeeSetEvent {
    pub fee: u64,
}

#[derive(Debug, PartialEq)]
pub struct SetMessageFeePayload {
    pub module: BytesN<32>,
    pub action: u8,
    pub chain: u16,
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
        validate_governance_header(&self.module, self.action, self.chain, ACTION_SET_MESSAGE_FEE)
    }
}

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
