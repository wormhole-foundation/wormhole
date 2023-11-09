use crate::{error::CoreBridgeError, state::EncodedVaa};
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
    #[account(
        mut,
        owner = crate::ID
    )]
    encoded_vaa: AccountInfo<'info>,
}

impl<'info> CloseEncodedVaa<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let acc_data = ctx.accounts.encoded_vaa.try_borrow_data()?;
        require!(
            acc_data.len() > 8
                && acc_data[..8] == <EncodedVaa as anchor_lang::Discriminator>::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        require_keys_eq!(
            EncodedVaa::write_authority_unsafe(&acc_data),
            ctx.accounts.write_authority.key(),
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
