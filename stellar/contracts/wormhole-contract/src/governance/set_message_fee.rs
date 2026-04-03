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

        if reader.remaining() != 0 {
            return Err(WormholeError::InvalidPayload);
        }

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Wormhole;
    use soroban_sdk::{Bytes, BytesN, IntoVal, Symbol, testutils::Events, vec};
    use wormhole_soroban_client::{CHAIN_ID_STELLAR, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, MODULE_CORE};

    fn build_payload(env: &Env, module: [u8; 32], action: u8, chain: u16, fee: u64) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &module));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[0u8; 24]));
        payload.append(&Bytes::from_slice(env, &fee.to_be_bytes()));
        payload
    }

    fn deploy_initialized(env: &Env) -> soroban_sdk::Address {
        let guardian = BytesN::<20>::from_array(env, &[0u8; 20]);
        let initial_guardians = vec![env, guardian];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        env.register(Wormhole, (initial_guardians, governance_emitter))
    }

    #[test]
    fn test_get_message_fee_default_zero() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        env.as_contract(&contract_id, || {
            assert_eq!(get_message_fee(&env), 0);
        });
    }

    #[test]
    fn test_set_message_fee_payload_parses_u256_padded_fee() {
        let env = Env::default();
        let payload = build_payload(
            &env,
            MODULE_CORE,
            ACTION_SET_MESSAGE_FEE,
            CHAIN_ID_STELLAR,
            1234,
        );

        let parsed = SetMessageFeePayload::try_from((&env, &payload)).unwrap();
        assert_eq!(parsed.action, ACTION_SET_MESSAGE_FEE);
        assert_eq!(parsed.chain, CHAIN_ID_STELLAR);
        assert_eq!(parsed.fee, 1234);
        assert_eq!(parsed.module, BytesN::<32>::from_array(&env, &MODULE_CORE));
    }

    #[test]
    fn test_set_message_fee_payload_rejects_short_payload() {
        let env = Env::default();
        let short = Bytes::from_slice(&env, &[0u8; 10]);
        let res = SetMessageFeePayload::try_from((&env, &short));
        assert_eq!(res, Err(WormholeError::InvalidPayload));
    }

    #[test]
    fn test_set_message_fee_validate_header_errors() {
        let env = Env::default();
        let bad_module = SetMessageFeePayload {
            module: BytesN::<32>::from_array(&env, &[9u8; 32]),
            action: ACTION_SET_MESSAGE_FEE,
            chain: CHAIN_ID_STELLAR,
            fee: 1,
        };
        assert_eq!(
            bad_module.validate(),
            Err(WormholeError::InvalidGovernanceModule)
        );

        let bad_action = SetMessageFeePayload {
            module: BytesN::<32>::from_array(&env, &MODULE_CORE),
            action: 99,
            chain: CHAIN_ID_STELLAR,
            fee: 1,
        };
        assert_eq!(
            bad_action.validate(),
            Err(WormholeError::InvalidGovernanceAction)
        );

        let bad_chain = SetMessageFeePayload {
            module: BytesN::<32>::from_array(&env, &MODULE_CORE),
            action: ACTION_SET_MESSAGE_FEE,
            chain: 2,
            fee: 1,
        };
        assert_eq!(
            bad_chain.validate(),
            Err(WormholeError::InvalidGovernanceChain)
        );
    }

    #[test]
    fn test_set_message_fee_execute_updates_state_and_event() {
        let env = Env::default();
        let contract_id = deploy_initialized(&env);
        let payload = SetMessageFeePayload {
            module: BytesN::<32>::from_array(&env, &MODULE_CORE),
            action: ACTION_SET_MESSAGE_FEE,
            chain: CHAIN_ID_STELLAR,
            fee: 777,
        };
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![&env],
            timestamp: 0,
            nonce: 0,
            emitter_chain: GOVERNANCE_CHAIN_ID,
            emitter_address: BytesN::from_array(&env, &[0u8; 32]),
            sequence: 0,
            consistency_level: wormhole_soroban_client::ConsistencyLevel::Confirmed,
            payload: Bytes::new(&env),
        };

        env.as_contract(&contract_id, || {
            SetMessageFeeAction::execute(&env, &vaa, &payload).unwrap();
            assert_eq!(get_message_fee(&env), 777);
        });

        let events = env.events().all().filter_by_contract(&contract_id);
        assert_eq!(
            events,
            vec![
                &env,
                (
                    contract_id,
                    vec![
                        &env,
                        Symbol::new(&env, "wormhole_core").into_val(&env),
                        Symbol::new(&env, "fee_set").into_val(&env),
                    ],
                    777u64.into_val(&env),
                )
            ]
        );
    }
}
