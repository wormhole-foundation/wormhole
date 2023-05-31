use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};
use core_bridge_program::types::{ChainId, ExternalAddress};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyTransferTokensArgs {
    pub nonce: u32,
    pub amount: u64,
    pub relayer_fee: u64,
    pub recipient: ExternalAddress,
    pub recipient_chain: ChainId,
}
