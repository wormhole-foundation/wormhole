use crate::{accounts, types::BridgeConfig, Initialize, InitializeData, MAX_LEN_GUARDIAN_KEYS};
use anchor_lang::{prelude::*, solana_program};

pub fn initialize(
    ctx: Context<Initialize>,
    len_guardians: u8,
    initial_guardian_key: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    config: BridgeConfig,
) -> ProgramResult {
    Ok(())
}
