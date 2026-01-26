//! Transfer fees governance action (`ACTION_TRANSFER_FEES`, value 4).
//!
//! Allows guardians to withdraw accumulated message fees from the contract.
//! Validates that sufficient balance remains (≥1 XLM) to prevent Stellar
//! account deallocation.

use crate::{
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    storage::StorageKey,
    utils::{address_from_ed25519_pk_bytes, get_native_token_address},
};
use core::convert::TryFrom;
use soroban_sdk::{Address, Bytes, BytesN, Env, contractevent, token};
use wormhole_soroban_client::{
    ACTION_TRANSFER_FEES, BytesReader, MINIMUM_CONTRACT_BALANCE, STORAGE_TTL_EXTENSION,
    STORAGE_TTL_THRESHOLD, TRANSFER_FEES_PAYLOAD_MIN_LENGTH, U256_PADDING_BYTES, VAA,
    WormholeError,
};

/// Event published when fees are transferred out.
///
/// Topics: ["wormhole_core", "fee_transfer"]
/// - "wormhole_core": Namespace for all core contract governance/lifecycle events
#[contractevent(topics = ["wormhole_core", "fee_transfer"])]
pub struct FeeTransferEvent {
    pub amount: u64,
    pub recipient: Address,
}

/// Parsed payload for transfer fees governance action.
#[derive(Debug, PartialEq)]
pub struct TransferFeesPayload {
    /// Module identifier (must be "Core").
    pub module: BytesN<32>,
    /// Governance action ID from the VAA header.
    ///
    /// Must equal `ACTION_TRANSFER_FEES` (4); any other value is rejected by
    /// `validate_governance_header`.
    pub action: u8,
    /// Target chain (0 for all, 61 for Stellar).
    pub chain: u16,
    /// Amount to transfer in stroops.
    pub amount: u64,
    /// Stellar address to receive the fees.
    pub recipient: Address,
}

impl<'a> TryFrom<(&'a Env, &'a Bytes)> for TransferFeesPayload {
    type Error = WormholeError;

    fn try_from(value: (&'a Env, &'a Bytes)) -> Result<Self, Self::Error> {
        let (env, payload) = value;

        if payload.len() < TRANSFER_FEES_PAYLOAD_MIN_LENGTH {
            return Err(WormholeError::InvalidPayload);
        }

        let mut reader = BytesReader::new(payload);
        let (module, action, chain) = parse_governance_header(env, &mut reader)?;

        reader.skip(U256_PADDING_BYTES)?;
        let amount = reader.read_u64_be()?;

        let recipient_pk_bytes: BytesN<32> = reader.read_bytes_n::<32>()?;
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
    fn validate(&self, env: &Env) -> Result<(), WormholeError> {
        validate_governance_header(&self.module, self.action, self.chain, ACTION_TRANSFER_FEES)?;

        let native_token_address = get_native_token_address(env);
        let token_client = token::TokenClient::new(env, &native_token_address);
        let contract_address = env.current_contract_address();
        let current_balance = u64::try_from(token_client.balance(&contract_address)).unwrap_or(0);

        if self.amount > current_balance {
            return Err(WormholeError::InsufficientFees);
        }

        let remaining_balance = current_balance.saturating_sub(self.amount);
        if remaining_balance < MINIMUM_CONTRACT_BALANCE {
            return Err(WormholeError::InsufficientFees);
        }

        Ok(())
    }
}

/// Returns the timestamp of the most recent fee transfer, if any.
pub fn get_last_fee_transfer(env: &Env) -> Option<u64> {
    env.storage().persistent().get(&StorageKey::LastFeeTransfer)
}

fn set_last_fee_transfer(env: &Env, timestamp: u64) {
    env.storage()
        .persistent()
        .set(&StorageKey::LastFeeTransfer, &timestamp);

    env.storage().persistent().extend_ttl(
        &StorageKey::LastFeeTransfer,
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

/// Governance action handler for fee transfers.
pub struct TransferFeesAction;

impl GovernanceAction for TransferFeesAction {
    type Payload = TransferFeesPayload;

    fn validate_payload(env: &Env, payload: &Self::Payload) -> Result<(), WormholeError> {
        payload.validate(env)
    }

    fn execute(env: &Env, _vaa: &VAA, payload: &Self::Payload) -> Result<(), WormholeError> {
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
