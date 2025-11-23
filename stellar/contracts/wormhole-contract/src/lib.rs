#![no_std]
#![allow(unused_variables)]

use soroban_sdk::{Address, Bytes, BytesN, Env, Vec, contract, contractimpl};
use wormhole_interface::{ConsistencyLevel, Error, GuardianSetInfo, VAA, WormholeCoreInterface};

#[contract]
pub struct Wormhole;

#[contractimpl]
impl WormholeCoreInterface for Wormhole {
    fn initialize(_env: Env, _initial_guardians: Vec<BytesN<20>>) -> Result<(), Error> {
        todo!("Implementation in later PRs")
    }

    fn is_initialized(_env: Env) -> bool {
        todo!()
    }

    fn verify_vaa(_env: Env, _vaa_bytes: Bytes) -> Result<bool, Error> {
        todo!()
    }

    fn parse_vaa(_env: Env, _vaa_bytes: Bytes) -> Result<VAA, Error> {
        todo!()
    }

    fn submit_contract_upgrade(_env: Env, _vaa_bytes: Bytes) -> Result<(), Error> {
        todo!()
    }

    fn submit_guardian_set_upgrade(_env: Env, _vaa_bytes: Bytes) -> Result<(), Error> {
        todo!()
    }

    fn submit_set_message_fee(_env: Env, _vaa_bytes: Bytes) -> Result<(), Error> {
        todo!()
    }

    fn submit_transfer_fees(_env: Env, _vaa_bytes: Bytes) -> Result<(), Error> {
        todo!()
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

    fn get_current_guardian_set_index(_env: Env) -> u32 {
        todo!()
    }

    fn get_guardian_set(_env: Env, _index: u32) -> Result<GuardianSetInfo, Error> {
        todo!()
    }

    fn get_guardian_set_expiry(_env: Env, _index: u32) -> Option<u64> {
        todo!()
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

    fn is_governance_vaa_consumed(_env: Env, _vaa_bytes: Bytes) -> Result<bool, Error> {
        todo!()
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
