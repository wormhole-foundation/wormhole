declare_program!(wormhole_post_message_shim);

use anchor_lang::{prelude::*, solana_program::bpf_loader_upgradeable};

use wormhole_post_message_shim::{
    cpi::accounts::PostMessage, program::WormholePostMessageShim, types::Finality,
};
use wormhole_solana_consts::{
    CORE_BRIDGE_CONFIG, CORE_BRIDGE_FEE_COLLECTOR, CORE_BRIDGE_PROGRAM_ID,
};

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(constraint = deployer.key() == program_data.upgrade_authority_address.unwrap_or_default())]
    deployer: Signer<'info>,

    #[account(
        seeds = [crate::ID.as_ref()],
        bump,
        seeds::program = bpf_loader_upgradeable::id(),
    )]
    program_data: Account<'info, ProgramData>,

    wormhole_post_message_shim: Program<'info, WormholePostMessageShim>,

    #[account(mut, address = CORE_BRIDGE_CONFIG)]
    /// CHECK: Wormhole bridge config. [`wormhole::post_message`] requires this account be mutable.
    pub bridge: UncheckedAccount<'info>,

    #[account(mut, seeds = [&emitter.key.to_bytes()], bump, seeds::program = wormhole_post_message_shim::ID)]
    /// CHECK: Wormhole Message. [`wormhole::post_message`] requires this account be signer and mutable.
    /// This program uses a PDA per emitter, since these are already bottle-necked by sequence and
    /// the bridge enforces that emitter must be identical for reused accounts.
    /// While this could be managed by the integrator, it seems more effective to have the shim manage these accounts.
    /// Bonus, this also allows Anchor to automatically handle deriving the address.
    pub message: UncheckedAccount<'info>,

    #[account(seeds = [b"emitter"], bump)]
    /// CHECK: Our emitter
    pub emitter: UncheckedAccount<'info>,

    #[account(mut)]
    /// CHECK: Emitter's sequence account. [`wormhole::post_message`] requires this account be mutable.
    /// Explicitly do not re-derive this account. The core bridge verifies the derivation anyway and
    /// as of Anchor 0.30.1, auto-derivation for other programs' accounts via IDL doesn't work.
    pub sequence: UncheckedAccount<'info>,

    #[account(mut, address = CORE_BRIDGE_FEE_COLLECTOR)]
    /// CHECK: Wormhole fee collector. [`wormhole::post_message`] requires this account be mutable.
    pub fee_collector: UncheckedAccount<'info>,

    /// Clock sysvar.
    pub clock: Sysvar<'info, Clock>,

    /// System program.
    pub system_program: Program<'info, System>,

    #[account(address = CORE_BRIDGE_PROGRAM_ID)]
    /// CHECK: Wormhole program.
    pub wormhole_program: UncheckedAccount<'info>,

    /// CHECK: Shim event authority
    pub wormhole_post_message_shim_ea: UncheckedAccount<'info>,
}

pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
    wormhole_post_message_shim::cpi::post_message(
        CpiContext::new_with_signer(
            ctx.accounts.wormhole_post_message_shim.to_account_info(),
            PostMessage {
                payer: ctx.accounts.payer.to_account_info(),
                bridge: ctx.accounts.bridge.to_account_info(),
                message: ctx.accounts.message.to_account_info(),
                emitter: ctx.accounts.emitter.to_account_info(),
                sequence: ctx.accounts.sequence.to_account_info(),
                fee_collector: ctx.accounts.fee_collector.to_account_info(),
                clock: ctx.accounts.clock.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
                wormhole_program: ctx.accounts.wormhole_program.to_account_info(),
                program: ctx.accounts.wormhole_post_message_shim.to_account_info(),
                event_authority: ctx.accounts.wormhole_post_message_shim_ea.to_account_info(),
            },
            &[&[b"emitter", &[ctx.bumps.emitter]]],
        ),
        0,
        Finality::Finalized,
        vec![],
    )?;

    Ok(())
}
