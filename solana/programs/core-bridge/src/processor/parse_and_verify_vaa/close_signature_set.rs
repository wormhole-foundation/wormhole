use crate::{
    legacy::utils::{AccountVariant, LegacyAnchorized},
    state::{PostedVaaV1, SignatureSet},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct CloseSignatureSet<'info> {
    #[account(mut)]
    sol_destination: Signer<'info>,

    /// Posted VAA.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            posted_vaa.message_hash().as_ref()
        ],
        bump,
        has_one = signature_set
    )]
    posted_vaa: Account<'info, LegacyAnchorized<PostedVaaV1>>,

    /// Signature set that may have been used to create the posted VAA account. If the `post_vaa_v1`
    /// instruction were used to create the posted VAA account, then the encoded signature set
    /// pubkey would be all zeroes.
    #[account(
        mut,
        close = sol_destination
    )]
    signature_set: Account<'info, AccountVariant<SignatureSet>>,
}

pub fn close_signature_set(_ctx: Context<CloseSignatureSet>) -> Result<()> {
    // Everything that has to be done is performed via account macros in the account context.
    Ok(())
}
