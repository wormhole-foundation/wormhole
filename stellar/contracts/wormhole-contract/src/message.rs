//! Cross-chain message posting and sequence management.
//!
//! Handles posting messages to be attested by Wormhole guardians, including
//! fee collection, sequence number management, and event emission.

use soroban_sdk::{Address, Bytes, BytesN, Env, address_payload::AddressPayload, contractevent, token};
use wormhole_soroban_client::{
    CHAIN_ID_STELLAR, ConsistencyLevel, PostedMessageData, STORAGE_TTL_EXTENSION,
    STORAGE_TTL_THRESHOLD, WormholeError,
};

use crate::{
    governance,
    storage::StorageKey,
    utils::{get_native_token_address, keccak256_hash},
};

/// Emitted when a cross-chain message is posted successfully.
///
/// Guardians observe these events and produce VAAs attesting to the message
/// contents. The event data matches the message body structure for easy
/// verification.
#[contractevent(topics = ["wormhole", "message_published"])]
struct MessagePublishedEvent {
    /// Caller-provided nonce for deduplication.
    nonce: u32,
    /// Auto-assigned sequence number for this emitter.
    sequence: u64,
    /// 32-byte contract ID of the emitter contract.
    emitter_address: BytesN<32>,
    /// Application-specific message data.
    payload: Bytes,
    /// Requested finality level for attestation.
    consistency_level: ConsistencyLevel,
}

/// Serializes message data to bytes for hashing (51-byte header + payload).
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

/// Returns the next sequence number for an emitter (0 if never used).
pub fn get_emitter_sequence(env: &Env, emitter: &Address) -> u64 {
    env.storage()
        .persistent()
        .get(&StorageKey::EmitterSequence(emitter.clone()))
        .unwrap_or(0)
}

/// Retrieves a previously posted message's hash by emitter and sequence.
pub fn get_posted_message_hash(env: &Env, emitter: &Address, sequence: u64) -> Option<BytesN<32>> {
    env.storage()
        .persistent()
        .get(&StorageKey::PostedMessage(emitter.clone(), sequence))
}

/// Atomically increments and returns the current sequence for an emitter.
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

/// Persists a message hash keyed by (emitter, sequence).
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

/// Posts a cross-chain message with optional fee collection.
///
/// Collects the configured message fee (if any) via `transfer_from`, assigns
/// a sequence number, hashes and stores the message, then emits a
/// `MessagePublishedEvent` for guardian observation.
///
/// # Errors
///
/// - `InsufficientFeePaid` if fee collection fails (requires prior approval)
pub fn post_message_with_fee(
    env: &Env,
    emitter: &Address,
    nonce: u32,
    payload: Bytes,
    consistency_level: ConsistencyLevel,
) -> Result<u64, WormholeError> {
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

    let emitter_bytes = match emitter.to_payload() {
        Some(AddressPayload::ContractIdHash(contract_id)) => contract_id,
        _ => return Err(WormholeError::InvalidEmitterAddress),
    };

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
