use crate::{
    state::{PostedVaaV1Bytes, SignatureSet},
    types::MessageHash,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
#[instruction(message_hash: MessageHash)]
pub struct ClosePostedVaaV1<'info> {
    #[account(mut)]
    sol_destination: Signer<'info>,

    #[account(
        mut,
        close = sol_destination,
        seeds = [PostedVaaV1Bytes::seed_prefix(), message_hash.as_ref()],
        bump
    )]
    posted_vaa: Account<'info, PostedVaaV1Bytes>,

    #[account(
        mut,
        close = sol_destination
    )]
    signature_set: Option<Account<'info, SignatureSet>>,
}

/// This directive acts as a placeholder in case we want to expand how posted VAAs are closed.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum ClosePostedVaaV1Directive {
    TryOnce,
}

pub fn close_posted_vaa_v1(
    ctx: Context<ClosePostedVaaV1>,
    directive: ClosePostedVaaV1Directive,
) -> Result<()> {
    match directive {
        ClosePostedVaaV1Directive::TryOnce => {
            msg!("Directive: TryOnce");
            let signature_set_key = ctx.accounts.posted_vaa.signature_set;
            match &ctx.accounts.signature_set {
                Some(signature_set) => require_keys_eq!(signature_set_key, signature_set.key()),
                None => require_keys_eq!(signature_set_key, Default::default()),
            };

            // Done.
            Ok(())
        }
    }
}
