use crate::{
    constants::*,
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    storage::StorageKey,
    utils::BytesReader,
};
use core::convert::TryFrom;
use soroban_sdk::{Bytes, BytesN, Env, contractevent};
use wormhole_interface::Error;

/// Event published when the message fee is updated.
///
/// Topics: ["wormhole_core", "fee_set"]
/// Data: fee (u64)
#[contractevent(topics = ["wormhole_core", "fee_set"], data_format = "single-value")]
struct MessageFeeSetEvent {
    fee: u64,
}

#[derive(Debug, PartialEq)]
pub struct SetMessageFeePayload {
    pub module: BytesN<32>,
    pub action: u8,
    pub chain: u16,
    pub fee: u64,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for SetMessageFeePayload {
    type Error = Error;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH {
            return Err(Error::InvalidPayload);
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
    fn validate(&self) -> Result<(), Error> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_SET_MESSAGE_FEE,
        )
    }
}

pub(crate) fn get_message_fee(env: &Env) -> u64 {
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

pub(crate) struct SetMessageFeeAction;

impl GovernanceAction for SetMessageFeeAction {
    type Payload = SetMessageFeePayload;

    fn validate_payload(_env: &Env, payload: &Self::Payload) -> Result<(), Error> {
        payload.validate()
    }

    fn execute(env: &Env, _vaa: &crate::vaa::VAA, payload: &Self::Payload) -> Result<(), Error> {
        set_message_fee(env, payload.fee);

        MessageFeeSetEvent { fee: payload.fee }.publish(env);

        Ok(())
    }
}
