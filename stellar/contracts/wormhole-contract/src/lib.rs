//! Wormhole Core Contract Implementation
//!
//! This crate contains the implementation of the Wormhole Core contract for Stellar/Soroban.
//! External users should depend on the `wormhole-interface` crate, not this implementation.
 
#![no_std]
#![allow(unused_variables)]
#![allow(unused_variables)]

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contract, contractimpl, token};
use wormhole_interface::{ConsistencyLevel, Error, GuardianSetInfo, VAA, WormholeCoreInterface};
use crate::governance::action::GovernanceAction;

pub mod constants;
pub mod storage;
pub mod utils;
pub mod vaa;
pub mod governance;
pub mod initialize;

#[contract]
pub struct Wormhole;

#[contractimpl]
impl WormholeCoreInterface for Wormhole {
    fn initialize(env: Env, initial_guardians: Vec<BytesN<20>>) -> Result<(), Error> {
        initialize::initialize(&env, initial_guardians)
    }

    fn is_initialized(env: Env) -> bool {
        initialize::is_initialized(&env)
    }

    /// Verify and validate a VAA (Verified Action Approval)
    /// Returns true if the VAA is valid and properly signed by the current guardian set
    fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<bool, Error> {
        vaa::verify_vaa(env, vaa_bytes)
    }

    /// Parse a VAA and return its components without verification
    /// Useful for inspection and debugging
    fn parse_vaa(
        env: Env,
        vaa_bytes: Bytes,
    ) -> Result<wormhole_interface::VAA, Error> {
        vaa::parse_vaa(&env, &vaa_bytes)
    }

    fn submit_contract_upgrade(_env: Env, _vaa_bytes: Bytes) -> Result<(), Error> {
        todo!()
    }

    /// Submit a guardian set upgrade governance VAA
    fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), Error> {
        governance::GuardianSetUpgradeAction::submit(&env, vaa_bytes)
    }

    /// Submit a set message fee governance VAA
    fn submit_set_message_fee(env: Env, vaa_bytes: Bytes) -> Result<(), Error> {
        governance::SetMessageFeeAction::submit(&env, vaa_bytes)
    }

    /// Submit a transfer fees governance VAA
    fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), Error> {
        governance::TransferFeesAction::submit(&env, vaa_bytes)
    }

    fn post_message(
        _env: Env,
        _emitter: Address,
        _nonce: u32,
        _payload: Bytes,
        _consistency_level: ConsistencyLevel,
    ) -> Result<u64, Error> {
        todo!()
    }

    /// Get the current guardian set index
    fn get_current_guardian_set_index(env: Env) -> u32 {
        governance::guardian_set::get_current_index(&env)
    }

    /// Get a guardian set by index
    fn get_guardian_set(
        env: Env,
        index: u32,
    ) -> Result<wormhole_interface::GuardianSetInfo, Error> {
        governance::guardian_set::get(&env, index)
    }

    /// Get the expiry time for a guardian set
    fn get_guardian_set_expiry(env: Env, index: u32) -> Option<u64> {
        governance::guardian_set::get_expiry(&env, index)
    }

    fn get_emitter_sequence(_env: Env, _emitter: Address) -> u64 {
        todo!()
    }

    fn get_posted_message_hash(_env: Env, _emitter: Address, _sequence: u64) -> Option<BytesN<32>> {
        todo!()
    }

    /// Get the current message fee in stroops
    fn get_message_fee(env: Env) -> u64 {
        governance::get_message_fee(&env)
    }

    /// Get the last fee transfer timestamp
    fn get_last_fee_transfer(env: Env) -> Option<u64> {
        governance::get_last_fee_transfer(&env)
    }

    /// Get the contract's current XLM balance
    fn get_contract_balance(env: Env) -> i128 {
        let native_token_address = utils::get_native_token_address(&env);
        let token_client = token::TokenClient::new(&env, &native_token_address);
        let contract_address = env.current_contract_address();
        token_client.balance(&contract_address)
    }

    /// Check if a governance VAA has been consumed (replay protection)
    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<bool, Error> {
        governance::is_governance_vaa_consumed(env, vaa_bytes)
    }

    /// Get the Wormhole chain ID for Stellar
    fn get_chain_id(_env: Env) -> u32 {
        u32::from(wormhole_interface::CHAIN_ID_STELLAR)
    }

}
