//! Transfer fees governance action (`ACTION_TRANSFER_FEES`, value 4).
//!
//! Allows guardians to withdraw accumulated message fees from the contract.

use crate::{
    governance::action::{GovernanceAction, parse_governance_header, validate_governance_header},
    utils::{address_from_ed25519_pk_bytes, get_native_token_address},
};
use core::convert::TryFrom;
use soroban_sdk::{Address, Bytes, BytesN, Env, contractevent, token};
use wormhole_soroban_client::{
    ACTION_TRANSFER_FEES, BytesReader, TRANSFER_FEES_PAYLOAD_MIN_LENGTH, U256_PADDING_BYTES, VAA,
    WormholeError,
};

/// Event published when fees are transferred out.
///
/// Topics: ["wormhole_core", "fee_transfer"]
/// - "wormhole_core": Namespace for all core contract governance/lifecycle
///   events
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

        if reader.remaining() != 0 {
            return Err(WormholeError::InvalidPayload);
        }

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

        Ok(())
    }
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

        FeeTransferEvent {
            amount: payload.amount,
            recipient: payload.recipient.clone(),
        }
        .publish(env);

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Wormhole;
    use soroban_sdk::{
        Address, Bytes, BytesN, Env, IntoVal, String, Symbol, Val, contract, contractimpl,
        contracttype,
        testutils::{Address as TestAddress, Events, Ledger},
    };
    use wormhole_soroban_client::{
        CHAIN_ID_STELLAR, GOVERNANCE_EMITTER, MODULE_CORE, NATIVE_TOKEN_ADDRESS,
    };

    #[contracttype]
    #[derive(Clone)]
    enum MockTokenStorage {
        Balance(Address),
    }

    #[contract]
    struct MockNativeToken;

    #[contractimpl]
    impl MockNativeToken {
        pub fn set_balance(env: Env, id: Address, amount: i128) {
            env.storage()
                .persistent()
                .set(&MockTokenStorage::Balance(id), &amount);
        }

        pub fn balance(env: Env, id: Address) -> i128 {
            env.storage()
                .persistent()
                .get(&MockTokenStorage::Balance(id))
                .unwrap_or(0)
        }

        pub fn transfer(env: Env, from: Address, to: Address, amount: i128) {
            let from_balance = Self::balance(env.clone(), from.clone());
            if from_balance < amount {
                panic!("insufficient");
            }
            let to_balance = Self::balance(env.clone(), to.clone());
            Self::set_balance(env.clone(), from, from_balance - amount);
            Self::set_balance(env, to, to_balance + amount);
        }
    }

    fn install_native_token_mock(env: &Env) -> (Address, MockNativeTokenClient<'_>) {
        let native = Address::from_string(&String::from_str(env, NATIVE_TOKEN_ADDRESS));
        env.register_at(&native, MockNativeToken, ());
        let client = MockNativeTokenClient::new(env, &native);
        (native, client)
    }

    fn deploy_initialized(env: &Env) -> Address {
        let guardian = BytesN::<20>::from_array(env, &[0u8; 20]);
        let initial_guardians = soroban_sdk::vec![env, guardian];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        env.register(Wormhole, (initial_guardians, governance_emitter))
    }

    fn build_payload(env: &Env, module: [u8; 32], action: u8, chain: u16, amount: u64) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &module));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &chain.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[0u8; 24]));
        payload.append(&Bytes::from_slice(env, &amount.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[7u8; 32]));
        payload
    }

    #[test]
    fn test_transfer_fees_payload_parses_amount_and_recipient() {
        let env = Env::default();
        let payload = build_payload(
            &env,
            MODULE_CORE,
            ACTION_TRANSFER_FEES,
            CHAIN_ID_STELLAR,
            12345,
        );
        let parsed = TransferFeesPayload::try_from((&env, &payload)).unwrap();

        assert_eq!(parsed.action, ACTION_TRANSFER_FEES);
        assert_eq!(parsed.chain, CHAIN_ID_STELLAR);
        assert_eq!(parsed.amount, 12345);
        assert_eq!(parsed.module, BytesN::from_array(&env, &MODULE_CORE));
    }

    #[test]
    fn test_transfer_fees_payload_rejects_short_payload() {
        let env = Env::default();
        let short = Bytes::from_slice(&env, &[0u8; 10]);
        let res = TransferFeesPayload::try_from((&env, &short));
        assert_eq!(res, Err(WormholeError::InvalidPayload));
    }

    #[test]
    fn test_transfer_fees_validate_header_errors() {
        let env = Env::default();
        let recipient = address_from_ed25519_pk_bytes(&env, &BytesN::from_array(&env, &[1u8; 32]));

        let bad_module = TransferFeesPayload {
            module: BytesN::from_array(&env, &[9u8; 32]),
            action: ACTION_TRANSFER_FEES,
            chain: CHAIN_ID_STELLAR,
            amount: 1,
            recipient: recipient.clone(),
        };
        assert_eq!(
            bad_module.validate(&env),
            Err(WormholeError::InvalidGovernanceModule)
        );

        let bad_action = TransferFeesPayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: 9,
            chain: CHAIN_ID_STELLAR,
            amount: 1,
            recipient: recipient.clone(),
        };
        assert_eq!(
            bad_action.validate(&env),
            Err(WormholeError::InvalidGovernanceAction)
        );

        let bad_chain = TransferFeesPayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_TRANSFER_FEES,
            chain: 2,
            amount: 1,
            recipient,
        };
        assert_eq!(
            bad_chain.validate(&env),
            Err(WormholeError::InvalidGovernanceChain)
        );
    }

    #[test]
    fn test_transfer_fees_validate_rejects_when_zero_balance() {
        let env = Env::default();
        let (_native_addr, _native_client) = install_native_token_mock(&env);
        let contract_id = deploy_initialized(&env);
        let recipient = <Address as TestAddress>::generate(&env);
        let payload = TransferFeesPayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_TRANSFER_FEES,
            chain: CHAIN_ID_STELLAR,
            amount: 1,
            recipient,
        };

        env.as_contract(&contract_id, || {
            let res = payload.validate(&env);
            assert_eq!(res, Err(WormholeError::InsufficientFees));
        });
    }

    #[test]
    fn test_transfer_fees_execute_moves_balance_and_emits_event() {
        let env = Env::default();
        env.ledger().set_timestamp(12345);
        let (_native_addr, native_client) = install_native_token_mock(&env);
        let contract_id = deploy_initialized(&env);
        let recipient = <Address as TestAddress>::generate(&env);
        let payload = TransferFeesPayload {
            module: BytesN::from_array(&env, &MODULE_CORE),
            action: ACTION_TRANSFER_FEES,
            chain: CHAIN_ID_STELLAR,
            amount: 5 * 10_000_000,
            recipient: recipient.clone(),
        };
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: soroban_sdk::vec![&env],
            timestamp: 0,
            nonce: 0,
            emitter_chain: 1,
            emitter_address: BytesN::from_array(&env, &[0u8; 32]),
            sequence: 0,
            consistency_level: wormhole_soroban_client::ConsistencyLevel::Confirmed,
            payload: Bytes::new(&env),
        };

        env.as_contract(&contract_id, || {
            native_client.set_balance(&contract_id, &(20 * 10_000_000i128));
            TransferFeesAction::execute(&env, &vaa, &payload).unwrap();

            assert_eq!(native_client.balance(&contract_id), 15 * 10_000_000i128);
            assert_eq!(native_client.balance(&recipient), 5 * 10_000_000i128);
        });

        let events = env.events().all().filter_by_contract(&contract_id);
        let amount_val: Val = (5 * 10_000_000u64).into_val(&env);
        let recipient_val: Val = recipient.into_val(&env);
        assert_eq!(
            events,
            soroban_sdk::vec![
                &env,
                (
                    contract_id,
                    soroban_sdk::vec![
                        &env,
                        Symbol::new(&env, "wormhole_core").into_val(&env),
                        Symbol::new(&env, "fee_transfer").into_val(&env),
                    ],
                    soroban_sdk::map![
                        &env,
                        (Symbol::new(&env, "amount"), amount_val),
                        (Symbol::new(&env, "recipient"), recipient_val),
                    ]
                    .into_val(&env),
                )
            ]
        );
    }
}
