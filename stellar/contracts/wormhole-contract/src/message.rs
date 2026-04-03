//! Cross-chain message posting and sequence management.
//!
//! Handles posting messages to be attested by Wormhole guardians, including
//! fee collection, sequence number management, and event emission.

use soroban_sdk::{
    Address, Bytes, BytesN, Env, address_payload::AddressPayload, contractevent, token,
};
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
        env.storage().persistent().extend_ttl(
            &StorageKey::MessageFee,
            STORAGE_TTL_THRESHOLD,
            STORAGE_TTL_EXTENSION,
        );

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

#[cfg(test)]
mod tests {
    use soroban_sdk::{
        Address, Bytes, BytesN, IntoVal, Symbol, Val,
        address_payload::AddressPayload,
        map,
        testutils::{Address as TestAddress, Events},
        vec,
    };

    use super::*;
    use crate::{Wormhole, WormholeClient};
    use wormhole_soroban_client::GOVERNANCE_EMITTER;

    fn deploy_initialized(env: &Env) -> (Address, WormholeClient<'_>) {
        let guardian = BytesN::from_array(env, &[0u8; 20]);
        let initial_guardians = vec![env, guardian];
        let governance_emitter = BytesN::from_array(env, &GOVERNANCE_EMITTER);
        let contract_id = env.register(Wormhole, (initial_guardians, governance_emitter));
        let client = WormholeClient::new(env, &contract_id);
        (contract_id, client)
    }

    fn address_to_bytes32(emitter: &Address) -> BytesN<32> {
        match emitter.to_payload() {
            Some(AddressPayload::ContractIdHash(contract_id)) => contract_id,
            _ => panic!("expected contract emitter address"),
        }
    }

    fn runtime_test_nonce(env: &Env) -> u32 {
        u32::try_from(env.ledger().timestamp()).unwrap_or_default()
    }

    #[test]
    fn test_post_message_accepts_generated_contract_emitter() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let emitter = <Address as TestAddress>::generate(&env);
        let nonce = runtime_test_nonce(&env);
        let res = client.try_post_message(
            &emitter,
            &nonce,
            &Bytes::from_array(&env, &[0xAA]),
            &ConsistencyLevel::Confirmed,
        );
        assert_eq!(res, Ok(Ok(0)));
    }

    #[test]
    fn test_post_message_rejects_ed25519_account() {
        let env = Env::default();
        let (_contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let pk_bytes = BytesN::from_array(&env, &[8u8; 32]);
        let emitter = crate::utils::address_from_ed25519_pk_bytes(&env, &pk_bytes);
        let nonce = runtime_test_nonce(&env);

        let res = client.try_post_message(
            &emitter,
            &nonce,
            &Bytes::from_array(&env, &[0xAA]),
            &ConsistencyLevel::Confirmed,
        );
        assert_eq!(res, Err(Ok(WormholeError::InvalidEmitterAddress)));
    }

    #[test]
    fn test_post_message_no_fee() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);
        assert_eq!(client.get_message_fee(), 0);

        env.mock_all_auths();

        // `post_message` only accepts contract emitters.
        let emitter = contract_id.clone();
        let nonce = runtime_test_nonce(&env);
        let payload = Bytes::from_array(&env, &[0xAA, 0xBB, 0xCC]);
        let cl = ConsistencyLevel::Confirmed;

        let ts_u32 = u32::try_from(env.ledger().timestamp()).unwrap_or(0);
        let emitter_bytes32 = address_to_bytes32(&emitter);

        let seq0 = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(seq0, 0);

        let events = env.events().all().filter_by_contract(&contract_id);
        assert_eq!(events.events().len(), 1);
        assert_eq!(client.get_emitter_sequence(&emitter), 1);

        let stored_hash0 = client
            .get_posted_message_hash(&emitter, &0)
            .expect("missing hash");

        let mut header = [0u8; 51];
        let mut off = 0usize;
        header[off..off + 4].copy_from_slice(&ts_u32.to_be_bytes());
        off += 4;
        header[off..off + 4].copy_from_slice(&nonce.to_be_bytes());
        off += 4;
        header[off..off + 2].copy_from_slice(&CHAIN_ID_STELLAR.to_be_bytes());
        off += 2;
        header[off..off + 32].copy_from_slice(&emitter_bytes32.to_array());
        off += 32;
        header[off..off + 8].copy_from_slice(&0u64.to_be_bytes());
        off += 8;
        header[off] = cl as u8;

        let mut msg_bytes = Bytes::from_array(&env, &header);
        msg_bytes.append(&payload);
        let expected_hash0 = keccak256_hash(&env, &msg_bytes);
        assert_eq!(stored_hash0, expected_hash0);

        let seq1 = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(seq1, 1);
        assert_eq!(client.get_emitter_sequence(&emitter), 2);
    }

    #[test]
    fn test_post_message_emits_event_payload() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let emitter = contract_id.clone();
        let emitter_bytes32 = address_to_bytes32(&emitter);
        let nonce = runtime_test_nonce(&env);
        let payload = Bytes::from_array(&env, &[0xAB, 0xCD]);
        let cl = ConsistencyLevel::Confirmed;

        let seq = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(seq, 0);

        let all_events = env.events().all();
        let contract_events = all_events.filter_by_contract(&contract_id);
        assert_eq!(contract_events.events().len(), 1);

        let consistency_level_val: Val = cl.into_val(&env);
        let emitter_val: Val = emitter_bytes32.into_val(&env);
        let nonce_val: Val = nonce.into_val(&env);
        let payload_val: Val = payload.clone().into_val(&env);
        let sequence_val: Val = 0u64.into_val(&env);

        assert_eq!(
            contract_events,
            vec![
                &env,
                (
                    contract_id.clone(),
                    vec![
                        &env,
                        Symbol::new(&env, "wormhole").into_val(&env),
                        Symbol::new(&env, "message_published").into_val(&env),
                    ],
                    map![
                        &env,
                        (
                            Symbol::new(&env, "consistency_level"),
                            consistency_level_val
                        ),
                        (Symbol::new(&env, "emitter_address"), emitter_val),
                        (Symbol::new(&env, "nonce"), nonce_val),
                        (Symbol::new(&env, "payload"), payload_val),
                        (Symbol::new(&env, "sequence"), sequence_val)
                    ]
                    .into_val(&env),
                )
            ]
        );
    }

    #[test]
    fn test_post_message_nonzero_fee_failure_does_not_mutate_state() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let fee: u64 = 1;
        env.as_contract(&contract_id, || {
            env.storage()
                .persistent()
                .set(&StorageKey::MessageFee, &fee);
        });
        assert_eq!(client.get_message_fee(), fee);

        let emitter = contract_id.clone();
        let nonce = runtime_test_nonce(&env);
        let res = client.try_post_message(
            &emitter,
            &nonce,
            &Bytes::from_array(&env, &[0x01]),
            &ConsistencyLevel::Confirmed,
        );
        assert_eq!(res, Err(Ok(WormholeError::InsufficientFeePaid)));
        assert_eq!(client.get_emitter_sequence(&emitter), 0);
        assert!(client.get_posted_message_hash(&emitter, &0).is_none());
    }

    #[test]
    fn test_post_message_fee_not_paid() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);

        let fee: u64 = 10_000;
        env.as_contract(&contract_id, || {
            env.storage()
                .persistent()
                .set(&StorageKey::MessageFee, &fee);
        });
        assert_eq!(client.get_message_fee(), fee);

        env.mock_all_auths();

        let emitter = <Address as TestAddress>::generate(&env);
        let nonce = runtime_test_nonce(&env);
        let payload = Bytes::from_array(&env, &[0x01, 0x02, 0x03]);
        let cl = ConsistencyLevel::Confirmed;

        let res = client.try_post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(res, Err(Ok(WormholeError::InsufficientFeePaid)));

        let events = env.events().all().filter_by_contract(&contract_id);
        assert_eq!(events.events().len(), 0);
    }

    #[test]
    fn test_emitter_sequence_increment() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let emitter = contract_id.clone();
        let nonce = runtime_test_nonce(&env);
        let payload = Bytes::from_array(&env, &[0xDE, 0xAD, 0xBE, 0xEF]);
        let cl = ConsistencyLevel::Confirmed;

        let s0 = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(s0, 0);
        assert_eq!(client.get_emitter_sequence(&emitter), 1);

        let s1 = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(s1, 1);
        assert_eq!(client.get_emitter_sequence(&emitter), 2);

        let s2 = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(s2, 2);
        assert_eq!(client.get_emitter_sequence(&emitter), 3);
    }

    #[test]
    fn test_posted_message_hash_stored() {
        let env = Env::default();
        let (contract_id, client) = deploy_initialized(&env);
        env.mock_all_auths();

        let emitter = contract_id.clone();
        let nonce = runtime_test_nonce(&env);
        let payload = Bytes::from_array(&env, &[0xAA, 0xBB, 0xCC]);
        let cl = ConsistencyLevel::Confirmed;
        let ts_u32 = u32::try_from(env.ledger().timestamp()).unwrap_or(0);

        let seq = client.post_message(&emitter, &nonce, &payload, &cl);
        assert_eq!(seq, 0);

        let stored = client
            .get_posted_message_hash(&emitter, &seq)
            .expect("missing stored hash");
        let emitter_bytes32 = address_to_bytes32(&emitter);

        let mut header = [0u8; 51];
        let mut off = 0usize;
        header[off..off + 4].copy_from_slice(&ts_u32.to_be_bytes());
        off += 4;
        header[off..off + 4].copy_from_slice(&nonce.to_be_bytes());
        off += 4;
        header[off..off + 2].copy_from_slice(&CHAIN_ID_STELLAR.to_be_bytes());
        off += 2;
        header[off..off + 32].copy_from_slice(&emitter_bytes32.to_array());
        off += 32;
        header[off..off + 8].copy_from_slice(&seq.to_be_bytes());
        off += 8;
        header[off] = cl as u8;

        let mut msg_bytes = Bytes::from_array(&env, &header);
        msg_bytes.append(&payload);
        let expected = keccak256_hash(&env, &msg_bytes);

        assert_eq!(stored, expected);
    }
}
