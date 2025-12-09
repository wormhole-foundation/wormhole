use soroban_sdk::{Address, Bytes, BytesN, Env, contractevent, token};
use wormhole_soroban_client::{
    CHAIN_ID_STELLAR, ConsistencyLevel, PostedMessageData, STORAGE_TTL_EXTENSION,
    STORAGE_TTL_THRESHOLD, WormholeError,
};

use crate::{
    governance, initialize,
    storage::StorageKey,
    utils::{address_to_bytes32, get_native_token_address, keccak256_hash},
};

/// Event published when a cross-chain message is posted.
///
/// Topics: ["wormhole", "message_published"]
/// - "wormhole": Namespace for cross-chain message events that guardians observe and attest to
/// - "message_published": Event type for new messages (guardians filter on this to find messages to sign)
#[contractevent(topics = ["wormhole", "message_published"])]
struct MessagePublishedEvent {
    nonce: u32,
    sequence: u64,
    emitter_address: BytesN<32>,
    payload: Bytes,
    consistency_level: ConsistencyLevel,
}

/// Serialize PostedMessageData for hashing
fn serialize_posted_message(
    message: &PostedMessageData,
    env: &Env,
) -> Result<Bytes, WormholeError> {
    // Build fixed-size header as a single array (51 bytes)
    let mut header = [0u8; 51];
    let mut offset = 0;

    // timestamp (4 bytes, big-endian)
    header[offset..offset + 4].copy_from_slice(&message.timestamp.to_be_bytes());
    offset += 4;

    // nonce (4 bytes, big-endian)
    header[offset..offset + 4].copy_from_slice(&message.nonce.to_be_bytes());
    offset += 4;

    // emitter_chain (2 bytes, big-endian u16)
    header[offset..offset + 2].copy_from_slice(&(message.emitter_chain as u16).to_be_bytes());
    offset += 2;

    // emitter_address (32 bytes)
    header[offset..offset + 32].copy_from_slice(&message.emitter_address.to_array());
    offset += 32;

    // sequence (8 bytes, big-endian)
    header[offset..offset + 8].copy_from_slice(&message.sequence.to_be_bytes());
    offset += 8;

    // consistency_level (1 byte)
    header[offset] = message.consistency_level as u8;

    // Create Bytes from header and append payload in a single operation
    let mut bytes = Bytes::from_array(env, &header);
    bytes.append(&message.payload);

    Ok(bytes)
}

pub fn get_emitter_sequence(env: &Env, emitter: &Address) -> u64 {
    env.storage()
        .persistent()
        .get(&StorageKey::EmitterSequence(emitter.clone()))
        .unwrap_or(0)
}

/// Get the hash of a posted message by emitter and sequence number.
/// Returns None if the message was not found.
pub fn get_posted_message_hash(env: &Env, emitter: &Address, sequence: u64) -> Option<BytesN<32>> {
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

pub fn post_message_with_fee(
    env: &Env,
    emitter: &Address,
    nonce: u32,
    payload: Bytes,
    consistency_level: ConsistencyLevel,
) -> Result<u64, WormholeError> {
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
                return Err(WormholeError::InsufficientFeePaid);
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
    let hash_bytes = keccak256_hash(env, &message_bytes);

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
