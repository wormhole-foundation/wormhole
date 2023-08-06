use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};
use solana_program::pubkey::Pubkey;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyTransferTokensWithPayloadArgs {
    pub nonce: u32,
    pub amount: u64,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub payload: Vec<u8>,
    pub cpi_program_id: Option<Pubkey>,
}
