use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyInitializeArgs {
    guardian_set_expiration_time: u32,
    fee: u64,
    initial_guardians: Vec<[u8; 20]>,
}
