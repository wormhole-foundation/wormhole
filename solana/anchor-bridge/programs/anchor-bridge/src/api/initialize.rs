use anchor_lang::{prelude::*, solana_program};

use crate::{
    accounts,
    anchor_bridge::Bridge,
    types::{BridgeConfig, Version},
    Initialize, InitializeData, MAX_LEN_GUARDIAN_KEYS,
};

pub fn initialize(
    ctx: Context<Initialize>,
    len_guardians: u8,
    initial_guardian_key: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    config: BridgeConfig,
) -> Result<Bridge, ProgramError> {
    Ok(Bridge {
        guardian_set_version: Version(0),
        config,
    })
}
