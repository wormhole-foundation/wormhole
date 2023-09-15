use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};
use solana_program::pubkey::Pubkey;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyInitializeArgs {
    _core_bridge: Pubkey,
}
