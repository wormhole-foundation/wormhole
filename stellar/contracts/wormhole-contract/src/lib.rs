#![no_std]

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contract, contractimpl};
use wormhole_soroban_client::{
    ConsistencyLevel, GuardianSetInfo, WormholeCoreInterface, WormholeError, CHAIN_ID_STELLAR, GOVERNANCE_CHAIN_ID, GOVERNANCE_EMITTER
};
use crate::governance::{
    guardian_set::get_current_guardian_set_index, guardian_set::get_guardian_set, guardian_set::get_guardian_set_expiry,
    ContractUpgradeAction, GovernanceAction, GuardianSetUpgradeAction, SetMessageFeeAction, TransferFeesAction,
};

mod governance;
mod initialize;
mod message;
mod storage;
mod utils;
mod vaa;

#[contract]
pub struct Wormhole;

#[contractimpl]
impl WormholeCoreInterface for Wormhole {
    /// Initialize the core contract with initial guardians and governance emitter. Can only be called once
    fn initialize(
        env: Env,
        initial_guardians: Vec<BytesN<20>>,
        governance_emitter: BytesN<32>,
    ) -> Result<(), WormholeError> {
        initialize::initialize(&env, initial_guardians, governance_emitter)
    }

    fn is_initialized(env: Env) -> bool {
        initialize::is_initialized(&env)
    }

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

    fn get_posted_message_hash(env: Env, emitter: Address, sequence: u64) -> Option<BytesN<32>> {
        message::get_posted_message_hash(&env, &emitter, sequence)
    }

    fn get_message_fee(env: Env) -> u64 {
        governance::get_message_fee(&env)
    }

    fn get_last_fee_transfer(env: Env) -> Option<u64> {
        governance::get_last_fee_transfer(&env)
    }

    fn get_contract_balance(env: Env) -> i128 {
        let native_token_address = utils::get_native_token_address(&env);
        let token_client = soroban_sdk::token::TokenClient::new(&env, &native_token_address);
        token_client.balance(&env.current_contract_address())
    }

    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        governance::is_governance_vaa_consumed(env, vaa_bytes)
    }

    fn get_chain_id() -> u32 {
        u32::from(CHAIN_ID_STELLAR)
    }

    fn get_governance_chain_id() -> u32 {
        GOVERNANCE_CHAIN_ID
    }

    fn get_governance_emitter(env: Env) -> BytesN<32> {
        env.storage()
            .persistent()
            .get(&storage::StorageKey::GovernanceEmitter)
            .unwrap_or_else(|| {
                // Fallback to default if not initialized
                BytesN::from_array(&env, &GOVERNANCE_EMITTER)
            })
    }
}
