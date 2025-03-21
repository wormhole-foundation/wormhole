use anchor_lang::prelude::*;
use wormhole_anchor_sdk::wormhole;
use wormhole_solana_consts::CORE_BRIDGE_PROGRAM_ID;

#[derive(Accounts)]
#[instruction(vaa_hash: [u8; 32])]
pub struct ConsumeCorePostedVaa<'info> {
    #[account(
        seeds = [
            wormhole::SEED_PREFIX_POSTED_VAA,
            &vaa_hash
        ],
        bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    /// CHECK: Verified Wormhole message account. The Wormhole program verified
    /// signatures and posted the account data here. Read-only.
    pub posted: UncheckedAccount<'info>,
}

pub fn consume_core_posted_vaa(
    ctx: Context<ConsumeCorePostedVaa>,
    _vaa_hash: [u8; 32],
) -> Result<()> {
    ctx.accounts.posted.try_borrow_data()?;
    Ok(())
}
