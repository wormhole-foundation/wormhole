//! Wormhole Core Contract implementation for Stellar/Soroban.
//!
//! This crate provides the complete implementation of the Wormhole Core
//! contract, enabling Stellar to participate in Wormhole's cross-chain
//! messaging protocol as Chain ID 61.
//!
//! # Architecture
//!
//! - [`Wormhole`] - Main contract struct implementing [`WormholeCoreInterface`]
//! - [`governance`] - Guardian-signed governance actions (upgrades, fees)
//! - [`vaa`] - VAA parsing and signature verification
//! - [`message`] - Cross-chain message posting
//! - [`storage`] - Persistent storage key definitions
//! - [`utils`] - Cryptographic and address conversion utilities

#![no_std]

use crate::governance::{
    ContractUpgradeAction, GovernanceAction, GuardianSetUpgradeAction, SetMessageFeeAction,
    TransferFeesAction,
    guardian_set::{get_current_guardian_set_index, get_guardian_set, get_guardian_set_expiry},
};
use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contract, contractimpl};
use storage::StorageKey;
use wormhole_soroban_client::{
    CHAIN_ID_STELLAR, ConsistencyLevel, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER, GuardianSetInfo,
    STORAGE_TTL_EXTENSION, STORAGE_TTL_THRESHOLD, WormholeCoreInterface, WormholeError,
    hash_address,
};

mod governance;
mod initialize;
mod message;
mod storage;
mod utils;
mod vaa;

/// Wormhole Core contract for Stellar/Soroban.
///
/// Implements the [`WormholeCoreInterface`] trait providing VAA verification,
/// governance action processing, and cross-chain message posting capabilities.
#[contract]
pub struct Wormhole;

/// Constructor and non-trait methods.
#[contractimpl]
impl Wormhole {
    /// Constructor called atomically during contract deployment.
    ///
    /// Sets up the initial guardian set (index 0) and stores the governance
    /// emitter address.
    ///
    /// # Arguments
    /// * `initial_guardians` - Ethereum addresses (20 bytes) of initial
    ///   guardians
    /// * `governance_emitter` - 32-byte address authorized to emit governance
    ///   VAAs
    ///
    /// # Panics
    /// Panics with `EmptyGuardianSet` if no guardians are provided.
    pub fn __constructor(
        env: Env,
        initial_guardians: Vec<BytesN<20>>,
        governance_emitter: BytesN<32>,
    ) {
        initialize::initialize_internal(&env, initial_guardians, governance_emitter);
    }
}

/// Implementation of the public Wormhole Core interface.
#[contractimpl]
impl WormholeCoreInterface for Wormhole {
    fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        vaa::verify_vaa(&env, &vaa_bytes)?;
        Ok(())
    }

    fn parse_vaa(
        env: Env,
        vaa_bytes: Bytes,
    ) -> Result<wormhole_soroban_client::VAA, WormholeError> {
        vaa::parse_vaa(&env, &vaa_bytes)
    }

    fn parse_and_verify_vaa(
        env: Env,
        vaa_bytes: Bytes,
    ) -> Result<wormhole_soroban_client::VAA, WormholeError> {
        vaa::parse_and_verify_vaa(&env, &vaa_bytes)
    }

    fn submit_contract_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        ContractUpgradeAction::submit(&env, vaa_bytes)
    }

    fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        GuardianSetUpgradeAction::submit(&env, vaa_bytes)
    }

    fn submit_set_message_fee(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        SetMessageFeeAction::submit(&env, vaa_bytes)
    }

    fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        TransferFeesAction::submit(&env, vaa_bytes)
    }

    fn post_message(
        env: Env,
        emitter: Address,
        nonce: u32,
        payload: Bytes,
        consistency_level: ConsistencyLevel,
    ) -> Result<u64, WormholeError> {
        emitter.require_auth();

        message::post_message_with_fee(&env, &emitter, nonce, payload, consistency_level)
    }

    fn get_current_guardian_set_index(env: Env) -> u32 {
        get_current_guardian_set_index(&env)
    }

    fn get_guardian_set(env: Env, index: u32) -> Result<GuardianSetInfo, WormholeError> {
        get_guardian_set(&env, index)
    }

    fn get_guardian_set_expiry(env: Env, index: u32) -> Option<u64> {
        get_guardian_set_expiry(&env, index)
    }

    fn get_emitter_sequence(env: Env, emitter: Address) -> u64 {
        message::get_emitter_sequence(&env, &emitter)
    }

    fn get_message_fee(env: Env) -> u64 {
        governance::get_message_fee(&env)
    }

    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, WormholeError> {
        governance::is_governance_vaa_consumed(env, vaa_bytes)
    }

    fn get_chain_id() -> u32 {
        u32::from(CHAIN_ID_STELLAR)
    }

    fn get_governance_chain_id() -> u32 {
        GOVERNANCE_CHAIN_ID
    }

    fn get_governance_emitter(env: Env) -> BytesN<32> {
        BytesN::from_array(&env, &GOVERNANCE_EMITTER)
    }

    fn record_address(env: Env, address: Address) -> Result<BytesN<32>, WormholeError> {
        let hash = hash_address(&env, &address);
        let key = StorageKey::AddressTable(hash.clone());
        env.storage().persistent().set(&key, &address);
        env.storage()
            .persistent()
            .extend_ttl(&key, STORAGE_TTL_THRESHOLD, STORAGE_TTL_EXTENSION);
        Ok(hash)
    }

    fn get_address_from_hash(env: Env, hash: BytesN<32>) -> Result<Address, WormholeError> {
        let key = StorageKey::AddressTable(hash);
        let Some(address) = env.storage().persistent().get(&key) else {
            return Err(WormholeError::AddressNotFound);
        };

        env.storage()
            .persistent()
            .extend_ttl(&key, STORAGE_TTL_THRESHOLD, STORAGE_TTL_EXTENSION);

        Ok(address)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::{Bytes, BytesN, Env, testutils::Address as _, vec};
    use wormhole_soroban_client::{
        ACTION_GUARDIAN_SET_UPGRADE, ACTION_SET_MESSAGE_FEE, ACTION_TRANSFER_FEES,
        GOVERNANCE_EMITTER, MODULE_CORE,
    };

    fn deploy_initialized(env: &Env) -> (Address, WormholeClient<'_>) {
        let guardian = BytesN::<20>::from_array(env, &[0u8; 20]);
        let initial_guardians = vec![env, guardian];
        let governance_emitter = BytesN::<32>::from_array(env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(Wormhole, (initial_guardians, governance_emitter));
        let client = WormholeClient::new(env, &contract_id);
        (contract_id, client)
    }

    fn build_unsigned_governance_vaa(
        env: &Env,
        emitter_chain: u16,
        emitter_address: [u8; 32],
        action: u8,
    ) -> Bytes {
        let mut payload = Bytes::new(env);
        payload.append(&Bytes::from_array(env, &MODULE_CORE));
        payload.push_back(action);
        payload.append(&Bytes::from_slice(env, &CHAIN_ID_STELLAR.to_be_bytes()));
        payload.append(&Bytes::from_array(env, &[0u8; 64]));

        // VAA bytes (version, guardian_set_index, sig_count=0, then body)
        let mut vaa = Bytes::new(env);
        vaa.push_back(1);
        vaa.append(&Bytes::from_slice(env, &0u32.to_be_bytes()));
        vaa.push_back(0);
        vaa.append(&Bytes::from_slice(env, &1u32.to_be_bytes())); // timestamp
        vaa.append(&Bytes::from_slice(env, &7u32.to_be_bytes())); // nonce
        vaa.append(&Bytes::from_slice(env, &emitter_chain.to_be_bytes()));
        vaa.append(&Bytes::from_array(env, &emitter_address));
        vaa.append(&Bytes::from_slice(env, &1u64.to_be_bytes())); // sequence
        vaa.push_back(ConsistencyLevel::Confirmed as u8);
        vaa.append(&payload);
        vaa
    }

    #[test]
    fn test_protocol_constant_getters() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        assert_eq!(client.get_chain_id(), u32::from(CHAIN_ID_STELLAR));
        assert_eq!(client.get_governance_chain_id(), GOVERNANCE_CHAIN_ID);
    }

    #[test]
    fn test_record_address_round_trip() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let address = Address::generate(&env);
        let hash = hash_address(&env, &address);

        assert_eq!(client.record_address(&address), hash);
        assert_eq!(client.get_address_from_hash(&hash), address);
    }

    #[test]
    fn test_record_account_address_round_trip() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let pk_bytes = BytesN::from_array(&env, &[7u8; 32]);
        let address = crate::utils::address_from_ed25519_pk_bytes(&env, &pk_bytes);
        let hash = hash_address(&env, &address);

        assert_eq!(client.record_address(&address), hash);
        assert_eq!(client.get_address_from_hash(&hash), address);
    }

    #[test]
    fn test_get_address_from_hash_returns_not_found_for_missing_hash() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let hash = BytesN::<32>::from_array(&env, &[9u8; 32]);

        assert_eq!(
            client.try_get_address_from_hash(&hash),
            Err(Ok(WormholeError::AddressNotFound))
        );
    }

    #[test]
    fn test_get_governance_emitter_returns_const() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let expected = BytesN::<32>::from_array(&env, &GOVERNANCE_EMITTER);
        assert_eq!(client.get_governance_emitter(), expected);
    }

    #[test]
    fn test_verify_vaa_wrapper_invalid_format() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let res = client.try_verify_vaa(&Bytes::new(&env));
        assert_eq!(res, Err(Ok(WormholeError::InvalidVAAFormat)));
    }

    #[test]
    fn test_is_governance_vaa_consumed_wrapper_invalid_format() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let res = client.try_is_governance_vaa_consumed(&Bytes::new(&env));
        assert_eq!(res, Err(Ok(WormholeError::InvalidVAAFormat)));
    }

    #[test]
    fn test_submit_set_message_fee_wrapper_invalid_governance_chain() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let vaa =
            build_unsigned_governance_vaa(&env, 9, GOVERNANCE_EMITTER, ACTION_SET_MESSAGE_FEE);
        let res = client.try_submit_set_message_fee(&vaa);
        assert_eq!(res, Err(Ok(WormholeError::InvalidGovernanceChain)));
    }

    #[test]
    fn test_submit_guardian_set_upgrade_wrapper_invalid_governance_emitter() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let mut bad_emitter = GOVERNANCE_EMITTER;
        bad_emitter[31] = 7;
        let vaa = build_unsigned_governance_vaa(
            &env,
            GOVERNANCE_CHAIN_ID as u16,
            bad_emitter,
            ACTION_GUARDIAN_SET_UPGRADE,
        );
        let res = client.try_submit_guardian_set_upgrade(&vaa);
        assert_eq!(res, Err(Ok(WormholeError::InvalidGovernanceEmitter)));
    }

    #[test]
    fn test_submit_transfer_fees_wrapper_invalid_governance_chain() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let vaa = build_unsigned_governance_vaa(&env, 9, GOVERNANCE_EMITTER, ACTION_TRANSFER_FEES);
        let res = client.try_submit_transfer_fees(&vaa);
        assert_eq!(res, Err(Ok(WormholeError::InvalidGovernanceChain)));
    }

    #[test]
    fn test_submit_contract_upgrade_wrapper_invalid_format() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        let res = client.try_submit_contract_upgrade(&Bytes::new(&env));
        assert_eq!(res, Err(Ok(WormholeError::InvalidVAAFormat)));
    }

    #[test]
    fn test_guardian_set_expiry_default_none() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        assert_eq!(client.get_guardian_set_expiry(&0), None);
    }
}
