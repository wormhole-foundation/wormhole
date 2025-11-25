use crate::{
    constants::*,
    governance, initialize,
    storage::StorageKey,
    utils::{address_to_bytes32, get_native_token_address},
};
use soroban_sdk::{Address, Bytes, BytesN, Env, contractevent, token};
use wormhole_interface::{ConsistencyLevel, Error, PostedMessageData};

/// Event published when a cross-chain message is posted.
///
/// Topics: ["wormhole", "message_published"]
/// Data: { nonce: u32, sequence: u64, emitter_address: BytesN<32>, payload: Bytes, consistency_level: ConsistencyLevel }
#[contractevent(topics = ["wormhole", "message_published"])]
struct MessagePublishedEvent {
    nonce: u32,
    sequence: u64,
    emitter_address: BytesN<32>,
    payload: Bytes,
    consistency_level: ConsistencyLevel,
}

/// Serialize PostedMessageData for hashing
#[allow(clippy::arithmetic_side_effects, clippy::cast_possible_truncation)]
fn serialize_posted_message(message: &PostedMessageData, env: &Env) -> Result<Bytes, Error> {
    let mut bytes = Bytes::new(env);

    for i in (0..4).rev() {
        bytes.push_back(((message.timestamp >> (i * 8)) & 0xFF) as u8);
    }

    for i in (0..4).rev() {
        bytes.push_back(((message.nonce >> (i * 8)) & 0xFF) as u8);
    }

    bytes.push_back((message.emitter_chain >> 8) as u8);
    bytes.push_back((message.emitter_chain & 0xFF) as u8);

    for byte in message.emitter_address.to_array().iter() {
        bytes.push_back(*byte);
    }

    for i in (0..8).rev() {
        bytes.push_back(((message.sequence >> (i * 8)) & 0xFF) as u8);
    }

    bytes.push_back(message.consistency_level as u8);

    bytes.append(&message.payload);

    Ok(bytes)
}

pub(crate) fn get_emitter_sequence(env: &Env, emitter: &Address) -> u64 {
    env.storage()
        .persistent()
        .get(&StorageKey::EmitterSequence(emitter.clone()))
        .unwrap_or(0)
}

/// Get the hash of a posted message by emitter and sequence number.
/// Returns None if the message was not found.
pub(crate) fn get_posted_message_hash(
    env: &Env,
    emitter: &Address,
    sequence: u64,
) -> Option<BytesN<32>> {
    env.storage()
        .persistent()
        .get(&StorageKey::PostedMessage(emitter.clone(), sequence))
}

fn next_emitter_sequence(env: &Env, emitter: &Address) -> u64 {
    let current = get_emitter_sequence(env, emitter);
    let next = current.saturating_add(1);
    env.storage()
        .persistent()
        .set(&StorageKey::EmitterSequence(emitter.clone()), &next);

    env.storage().persistent().extend_ttl(
        &StorageKey::EmitterSequence(emitter.clone()),
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );

    current
}

fn store_posted_message(env: &Env, emitter: &Address, sequence: u64, message_hash: &BytesN<32>) {
    env.storage().persistent().set(
        &StorageKey::PostedMessage(emitter.clone(), sequence),
        message_hash,
    );

    env.storage().persistent().extend_ttl(
        &StorageKey::PostedMessage(emitter.clone(), sequence),
        STORAGE_TTL_THRESHOLD,
        STORAGE_TTL_EXTENSION,
    );
}

pub(crate) fn post_message_with_fee(
    env: &Env,
    emitter: &Address,
    nonce: u32,
    payload: Bytes,
    consistency_level: ConsistencyLevel,
) -> Result<u64, Error> {
    initialize::require_initialized(env)?;

    let required_fee = governance::get_message_fee(env);

    if required_fee > 0 {
        let native_token = get_native_token_address(env);
        let token_client = token::TokenClient::new(env, &native_token);
        let contract = env.current_contract_address();

        match token_client.try_transfer_from(
            &contract,
            emitter,
            &contract,
            &i128::from(required_fee),
        ) {
            Ok(Ok(())) => {}
            _ => {
                return Err(Error::InsufficientFeePaid);
            }
        }
    }

    let sequence = next_emitter_sequence(env, emitter);

    let emitter_bytes = address_to_bytes32(env, emitter);

    let message_data = PostedMessageData {
        timestamp: u32::try_from(env.ledger().timestamp()).unwrap_or(0),
        nonce,
        emitter_chain: u32::from(CHAIN_ID_STELLAR),
        emitter_address: emitter_bytes.clone(),
        sequence,
        consistency_level,
        payload: payload.clone(),
    };

    let message_bytes = serialize_posted_message(&message_data, env)?;
    let hash_bytes = crate::utils::keccak256_hash(env, &message_bytes);

    store_posted_message(env, emitter, sequence, &hash_bytes);

    MessagePublishedEvent {
        nonce,
        sequence,
        emitter_address: emitter_bytes,
        payload,
        consistency_level,
    }
    .publish(env);

    Ok(sequence)
}
