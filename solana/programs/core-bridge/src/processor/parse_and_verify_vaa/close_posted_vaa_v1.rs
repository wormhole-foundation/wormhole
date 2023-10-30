use crate::{
    error::CoreBridgeError,
    legacy::utils::{AccountVariant, LegacyAnchorized},
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
    posted_vaa: Account<'info, LegacyAnchorized<PostedVaaV1>>,

    /// Signature set that may have been used to create the posted VAA account. If the `post_vaa_v1`
    /// instruction were used to create the posted VAA account, then the encoded signature set
    /// pubkey would be all zeroes.
    #[account(
        mut,
        close = sol_destination
    )]
    signature_set: Option<Account<'info, AccountVariant<SignatureSet>>>,
}

pub fn close_posted_vaa_v1(ctx: Context<ClosePostedVaaV1>) -> Result<()> {
    let verified_signature_set = ctx.accounts.posted_vaa.signature_set;
    match &ctx.accounts.signature_set {
        Some(signature_set) => {
            // Verify that the signature set pubkey in the posted VAA account equals the signature
            // set pubkey passed into the account context.
            require_keys_eq!(
                signature_set.key(),
                verified_signature_set,
                CoreBridgeError::InvalidSignatureSet
            );
        }
        None => {
            // If there were no signature set used when this posted VAA was created (using the
            // `post_vaa_v1` instruction), verify that there is actually no signature set pubkey
            // written to this account.
            require_keys_eq!(
                verified_signature_set,
                Pubkey::default(),
                ErrorCode::AccountNotEnoughKeys
            );
        }
    }

    // Done.
    Ok(())
}
