use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAnchorized,
    state::{PostedVaaV1, SignatureSet},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct ClosePostedVaaV1<'info> {
    #[account(mut)]
    sol_destination: Signer<'info>,

    /// Posted VAA.
    #[account(
        mut,
        close = sol_destination,
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            posted_vaa.message_hash().as_ref()
        ],
        bump,
    )]
    posted_vaa: Account<'info, LegacyAnchorized<4, PostedVaaV1>>,

    #[account(
        mut,
        close = sol_destination
    )]
    signature_set: Option<Account<'info, LegacyAnchorized<0, SignatureSet>>>,
}

pub fn close_posted_vaa_v1(ctx: Context<ClosePostedVaaV1>) -> Result<()> {
    let verified_signature_set = ctx.accounts.posted_vaa.signature_set;
    match &ctx.accounts.signature_set {
        Some(signature_set) => {
            require_keys_eq!(
                signature_set.key(),
                verified_signature_set,
                CoreBridgeError::InvalidSignatureSet
            )
        }
        None => require_keys_eq!(
            verified_signature_set,
            Pubkey::default(),
            ErrorCode::AccountNotEnoughKeys
        ),
    };

    // Done.
    Ok(())
}
