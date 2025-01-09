use anchor_lang::prelude::*;

use crate::state::GuardianSignatures;

#[derive(Accounts)]
pub struct CloseSignatures<'info> {
    #[account(mut, has_one = refund_recipient, close = refund_recipient)]
    guardian_signatures: Account<'info, GuardianSignatures>,

    #[account(mut, address = guardian_signatures.refund_recipient)]
    refund_recipient: Signer<'info>,
}

/// Allows the initial payer to close the signature account in case the query was invalid.
pub fn close_signatures(_ctx: Context<CloseSignatures>) -> Result<()> {
    Ok(())
}
