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
    let version = Version(0);

    // Initialize the Guardian Set for the first time.
    ctx.accounts.guardian_set.version = version;
    ctx.accounts.guardian_set.creation_time = ctx.accounts.clock.unix_timestamp as u32;
    ctx.accounts.guardian_set.keys = initial_guardian_key;
    ctx.accounts.guardian_set.len_keys = len_guardians;

    // Generate a Version 0 state for the bridges genesis.
    Ok(Bridge {
        guardian_set_version: version,
        config,
    })
}
