mod native;
pub use native::*;

use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct MockLegacyTransferTokensArgs {
    pub nonce: u32,
    pub amount: u64,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
}
