use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};
use core_bridge_program::types::{ChainId, ExternalAddress};
use solana_program::pubkey::Pubkey;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyTransferTokensWithPayloadArgs {
    pub nonce: u32,
    pub amount: u64,
    pub redeemer: ExternalAddress,
    pub redeemer_chain: ChainId,
    pub payload: Vec<u8>,
    pub cpi_program_id: Option<Pubkey>,
}
