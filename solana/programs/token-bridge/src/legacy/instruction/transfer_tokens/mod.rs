mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TransferTokensArgs {
    pub nonce: u32,
    pub amount: u64,
    pub relayer_fee: u64,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
}
