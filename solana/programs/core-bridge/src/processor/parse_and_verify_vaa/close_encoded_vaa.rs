use crate::{error::CoreBridgeError, zero_copy::EncodedVaa};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct CloseEncodedVaa<'info> {
    /// This account is only required to be mutable for the `CloseVaaAccount` directive. This
    /// authority is the same signer that originally created the VAA accounts, so he is the one that
    /// will receive the lamports back for the closed accounts.
    #[account(mut)]
    write_authority: Signer<'info>,

    /// CHECK: The encoded VAA account, which stores the VAA buffer. This buffer must first be
    /// written to and then verified.
    #[account(mut)]
    encoded_vaa: AccountInfo<'info>,
}

impl<'info> CloseEncodedVaa<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Check write authority.
        let vaa = EncodedVaa::parse_unverified(&ctx.accounts.encoded_vaa)?;
        require_keys_eq!(
            ctx.accounts.write_authority.key(),
            vaa.write_authority(),
            CoreBridgeError::WriteAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

#[access_control(CloseEncodedVaa::constraints(&ctx))]
pub fn close_encoded_vaa(ctx: Context<CloseEncodedVaa>) -> Result<()> {
    crate::utils::close_account(&ctx.accounts.encoded_vaa, &ctx.accounts.write_authority)
}
