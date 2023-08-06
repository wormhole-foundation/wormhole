use crate::{constants::UPGRADE_SEED_PREFIX, state::Config, ID};
use anchor_lang::prelude::*;
use core_bridge_program::CoreBridge;
use solana_program::{bpf_loader_upgradeable, program::invoke};
use wormhole_solana_common::{BpfLoaderUpgradeable, SeedPrefix};

#[derive(Accounts)]
pub struct Initialize<'info> {
    /// This account should be the same payer that deployed the Core Bridge BPF.
    #[account(mut)]
    deployer: Signer<'info>,

    /// Core Bridge data and config. This account is necessary to publish Wormhole messages and
    /// redeem governance VAAs.
    #[account(
        init,
        payer = deployer,
        space = Config::INIT_SPACE,
        seeds = [Config::seed_prefix()],
        bump,
    )]
    config: Account<'info, Config>,

    /// CHECK: Before we initialize this program, we need to verify that the upgrade authority is
    /// set for this program. To do so, call the `SetAuthority` instruction via the
    /// BpfLoaderUpgradeable native program.
    #[account(
        seeds = [UPGRADE_SEED_PREFIX],
        bump,
    )]
    upgrade_authority: AccountInfo<'info>,

    /// CHECK: BPF Loader Upgradeable program needs to modify this program's data to change the
    /// upgrade authority. We check this PDA address just in case there is another program that this
    /// deployer has deployed.
    ///
    /// NOTE: Set upgrade authority is scary because any public key can be used to set as the
    /// authority.
    #[account(
        mut,
        seeds = [ID.as_ref()],
        bump,
        seeds::program = bpf_loader_upgradeable_program,
    )]
    program_data: AccountInfo<'info>,

    system_program: Program<'info, System>,
    bpf_loader_upgradeable_program: Program<'info, BpfLoaderUpgradeable>,
    core_bridge_program: Program<'info, CoreBridge>,
}

pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
    // Set config data.
    //
    // NOTE: This config account is pointless and is never used in any of the instruction handlers.
    ctx.accounts.config.set_inner(Config {
        core_bridge: CoreBridge::id(),
    });

    // Finally set the upgrade authority to this program's PDA address.
    invoke(
        &bpf_loader_upgradeable::set_upgrade_authority(
            &ID,
            &ctx.accounts.deployer.key(),
            Some(&ctx.accounts.upgrade_authority.key()),
        ),
        &ctx.accounts.to_account_infos(),
    )
    .map_err(Into::into)
}
