use crate::{
    constants::*,
    governance::action::{parse_governance_header, validate_governance_header, GovernanceAction},
    storage::StorageKey,
    utils::{address_from_ed25519_pk_bytes, get_native_token_address, BytesReader},
};
use core::convert::TryFrom;
use soroban_sdk::{contractevent, token, Address, Bytes, BytesN, Env};
use wormhole_interface::Error;

/// Event published when fees are transferred out.
///
/// Topics: ["wormhole_core", "fee_xfer"]
/// Data: { amount: u64, recipient: Address }
#[contractevent(topics = ["wormhole_core", "fee_xfer"])]
struct FeeTransferEvent {
    amount: u64,
    recipient: Address,
}

#[derive(Debug, PartialEq)]
pub struct TransferFeesPayload {
    pub module: BytesN<32>,
    pub action: u8,
    pub chain: u16,
    pub amount: u64,
    pub recipient: Address,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for TransferFeesPayload {
    type Error = Error;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < TRANSFER_FEES_PAYLOAD_MIN_LENGTH {
            return Err(Error::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;

        reader.skip(U256_PADDING_BYTES)?;
        let amount = reader.read_u64_be()?;

        let recipient_pk_bytes = reader.read_bytes_n::<32>(env)?;
        let recipient = address_from_ed25519_pk_bytes(env, &recipient_pk_bytes);

        Ok(TransferFeesPayload {
            module,
            action,
            chain,
            amount,
            recipient,
        })
    }
}

impl TransferFeesPayload {
    fn validate(&self, env: &Env) -> Result<(), Error> {
        validate_governance_header(
            &self.module,
            self.action,
            self.chain,
            ACTION_TRANSFER_FEES,
        )?;

        let native_token_address = get_native_token_address(env);
        let token_client = token::TokenClient::new(env, &native_token_address);
        let contract_address = env.current_contract_address();
        let current_balance = u64::try_from(token_client.balance(&contract_address))
            .unwrap_or(0);

        if self.amount > current_balance {
            return Err(Error::InsufficientFees);
        }

        let remaining_balance = current_balance.saturating_sub(self.amount);
        if remaining_balance < MINIMUM_CONTRACT_BALANCE {
            return Err(Error::InsufficientFees);
        }

        Ok(())
    }
}

pub(crate) fn get_last_fee_transfer(env: &Env) -> Option<u64> {
    env.storage().persistent().get(&StorageKey::LastFeeTransfer)
}


fn set_last_fee_transfer(env: &Env, timestamp: u64) {
    env.storage()
        .persistent()
        .set(&StorageKey::LastFeeTransfer, &timestamp);

    env.storage()
        .persistent()
        .extend_ttl(&StorageKey::LastFeeTransfer, STORAGE_TTL_THRESHOLD, STORAGE_TTL_EXTENSION);
}

pub(crate) struct TransferFeesAction;

impl GovernanceAction for TransferFeesAction {
    type Payload = TransferFeesPayload;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), Error> {
        payload.validate(env)
    }

    fn execute(env: &Env, _vaa: &crate::vaa::VAA, payload: &Self::Payload) -> Result<(), Error> {
        let native_token_address = get_native_token_address(env);
        let token_client = token::TokenClient::new(env, &native_token_address);
        let contract_address = env.current_contract_address();

        token_client.transfer(
            &contract_address,
            &payload.recipient,
            &i128::from(payload.amount),
        );

        set_last_fee_transfer(env, env.ledger().timestamp());

        FeeTransferEvent {
            amount: payload.amount,
            recipient: payload.recipient.clone(),
        }
        .publish(env);

        Ok(())
    }
}

