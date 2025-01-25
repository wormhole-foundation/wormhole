use anchor_lang::prelude::*;

use crate::state::GuardianSignatures;

#[derive(Accounts)]
pub struct CloseSignatures<'info> {
    #[account(mut, has_one = refund_recipient, close = refund_recipient)]
    guardian_signatures: Account<'info, GuardianSignatures>,

    #[account(mut, address = guardian_signatures.refund_recipient)]
    refund_recipient: Signer<'info>,
}

pub fn close_signatures(_ctx: Context<CloseSignatures>) -> Result<()> {
    Ok(())
}
