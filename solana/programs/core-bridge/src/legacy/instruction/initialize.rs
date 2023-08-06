use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyInitializeArgs {
    pub guardian_set_ttl_seconds: u32,
    pub fee_lamports: u64,
    pub initial_guardians: Vec<[u8; 20]>,
}
