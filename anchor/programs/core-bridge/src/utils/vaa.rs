use crate::types::Timestamp;
use anchor_lang::solana_program::keccak;

#[inline]
pub fn quorum(num_guardians: usize) -> usize {
    (2 * num_guardians) / 3 + 1
}

#[inline]
pub fn compute_message_hash(
    timestamp: Timestamp,
    nonce: u32,
    emitter_chain: u16,
    emitter_address: &[u8; 32],
    sequence: u64,
    consistency_level: u8,
    payload: &[u8],
) -> keccak::Hash {
    let mut body = Vec::with_capacity(51 + payload.len());
    body.extend_from_slice(&timestamp.to_be_bytes());
    body.extend_from_slice(&nonce.to_be_bytes());
    body.extend_from_slice(&emitter_chain.to_be_bytes());
    body.extend_from_slice(emitter_address.as_ref());
    body.extend_from_slice(&sequence.to_be_bytes());
    body.push(consistency_level);
    body.extend_from_slice(payload);

    keccak::hash(&body)
}
