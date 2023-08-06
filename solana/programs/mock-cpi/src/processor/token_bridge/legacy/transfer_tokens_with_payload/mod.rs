mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::*;

pub const CUSTOM_SENDER_SEED_PREFIX: &[u8] = b"custom_sender_authority";

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct MockLegacyTransferTokensWithPayloadArgs {
    pub nonce: u32,
    pub amount: u64,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub payload: Vec<u8>,
}
