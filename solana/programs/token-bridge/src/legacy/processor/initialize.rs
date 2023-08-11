use crate::{legacy::instruction::LegacyInitializeArgs, state::Config};
use anchor_lang::prelude::*;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init,
        payer = payer,
        space = Config::INIT_SPACE,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, Config>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

pub fn initialize(ctx: Context<Initialize>, _args: LegacyInitializeArgs) -> Result<()> {
    // NOTE: This config account is pointless and is never used in any of the instruction handlers.
    ctx.accounts.config.set_inner(Config {
        core_bridge_program: core_bridge_program::ID,
    });

    // Done.
    Ok(())
}
