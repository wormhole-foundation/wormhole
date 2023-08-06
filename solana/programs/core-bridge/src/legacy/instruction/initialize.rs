use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// Arguments used to initialize the Core Bridge program.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitializeArgs {
    pub guardian_set_ttl_seconds: u32,
    pub fee_lamports: u64,
    pub initial_guardians: Vec<[u8; 20]>,
}
