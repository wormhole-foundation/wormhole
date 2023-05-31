use crate::types::{ChainId, ExternalAddress, Finality, MessageHash, Timestamp};
use anchor_lang::solana_program::keccak;

#[inline]
pub fn quorum(num_guardians: usize) -> usize {
    (2 * num_guardians) / 3 + 1
}

#[inline]
pub fn compute_message_hash(
    timestamp: Timestamp,
    nonce: u32,
    emitter_chain: ChainId,
    emitter_address: &ExternalAddress,
    sequence: u64,
    finality: Finality,
    payload: &[u8],
) -> MessageHash {
    let mut body = Vec::with_capacity(51 + payload.len());
    body.extend_from_slice(&timestamp.to_be_bytes());
    body.extend_from_slice(&nonce.to_be_bytes());
    body.extend_from_slice(&emitter_chain.to_be_bytes());
    body.extend_from_slice(emitter_address.as_ref());
    body.extend_from_slice(&sequence.to_be_bytes());
    body.push(finality.into());
    body.extend_from_slice(payload);

    keccak::hash(&body).into()
}
