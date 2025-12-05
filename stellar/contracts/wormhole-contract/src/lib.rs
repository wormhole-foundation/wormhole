#![no_std]

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contract, contractimpl};
use wormhole_soroban_client::{
    ConsistencyLevel, GuardianSetInfo, WormholeCoreInterface, WormholeError,
};

mod governance;
mod initialize;
mod storage;
mod utils;
mod vaa;

use governance::GovernanceAction;

#[contract]
pub struct Wormhole;

#[contractimpl]
impl WormholeCoreInterface for Wormhole {
    fn initialize(env: Env, initial_guardians: Vec<BytesN<20>>) -> Result<(), WormholeError> {
        initialize::initialize(&env, initial_guardians)
    }

    fn is_initialized(env: Env) -> bool {
        initialize::is_initialized(&env)
    }

    fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        vaa::verify_vaa(&env, &vaa_bytes)?;
        Ok(())
    }

    fn parse_vaa(env: Env, vaa_bytes: Bytes) -> Result<wormhole_soroban_client::VAA, WormholeError> {
        vaa::parse_vaa(&env, &vaa_bytes)
    }

    fn submit_contract_upgrade(_env: Env, _vaa_bytes: Bytes) -> Result<(), WormholeError> {
        todo!()
    }

    fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        governance::GuardianSetUpgradeAction::submit(&env, vaa_bytes)
    }

    fn submit_set_message_fee(_env: Env, _vaa_bytes: Bytes) -> Result<(), WormholeError> {
        todo!()
    }

    fn submit_transfer_fees(_env: Env, _vaa_bytes: Bytes) -> Result<(), WormholeError> {
        todo!()
    }

    fn post_message(
        _env: Env,
        _emitter: Address,
        _nonce: u32,
        _payload: Bytes,
        _consistency_level: ConsistencyLevel,
    ) -> Result<u64, WormholeError> {
        todo!()
    }

    fn get_current_guardian_set_index(env: Env) -> u32 {
        governance::guardian_set::get_current_index(&env)
    }

    fn get_guardian_set(env: Env, index: u32) -> Result<GuardianSetInfo, WormholeError> {
        governance::guardian_set::get(&env, index)
    }

    fn get_guardian_set_expiry(env: Env, index: u32) -> Option<u64> {
        governance::guardian_set::get_expiry(&env, index)
    }

    fn get_emitter_sequence(_env: Env, _emitter: Address) -> u64 {
        todo!()
    }

    fn get_posted_message_hash(_env: Env, _emitter: Address, _sequence: u64) -> Option<BytesN<32>> {
        todo!()
    }

    fn get_message_fee(_env: Env) -> u64 {
        todo!()
    }

    fn get_last_fee_transfer(_env: Env) -> Option<u64> {
        todo!()
    }

    fn get_contract_balance(_env: Env) -> i128 {
        todo!()
    }

    fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError> {
        if governance::is_governance_vaa_consumed(env, vaa_bytes)? {
            Err(WormholeError::GovernanceVAAAlreadyConsumed)
        } else {
            Ok(())
        }
    }

    fn get_chain_id(_env: Env) -> u32 {
        todo!()
    }

    fn get_governance_chain_id(_env: Env) -> u32 {
        todo!()
    }

    fn get_governance_emitter(_env: Env) -> BytesN<32> {
        todo!()
    }
}
