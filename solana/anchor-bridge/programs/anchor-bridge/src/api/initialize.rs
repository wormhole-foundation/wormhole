use anchor_lang::{prelude::*, solana_program};

use crate::{
    Result,
    accounts,
    anchor_bridge::Bridge,
    types::{BridgeConfig, Index},
    Initialize,
    InitializeData,
    MAX_LEN_GUARDIAN_KEYS,
};

pub fn initialize(
    ctx: Context<Initialize>,
    len_guardians: u8,
    initial_guardian_key: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    config: BridgeConfig,
) -> Result<Bridge> {
    let index = Index(0);

    // Initialize the Guardian Set for the first time.
    ctx.accounts.guardian_set.index = index;
    ctx.accounts.guardian_set.creation_time = ctx.accounts.clock.unix_timestamp as u32;
    ctx.accounts.guardian_set.keys = initial_guardian_key;
    ctx.accounts.guardian_set.len_keys = len_guardians;

    // Create an initial bridge state, labeled index 0.
    Ok(Bridge {
        guardian_set_index: index,
        config,
    })
}
