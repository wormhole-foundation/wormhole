use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct InitAndProcessMessageV1<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Draft message.
    #[account(
        mut,
        seeds = [b"draft_message"],
        bump
    )]
    message: AccountInfo<'info>,
}
