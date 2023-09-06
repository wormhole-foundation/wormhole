use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAnchorized,
    state::{EncodedVaa, PostedVaaV1, PostedVaaV1Info, ProcessingStatus},
    types::VaaVersion,
};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::Vaa;

#[derive(Accounts)]
pub struct PostVaaV1<'info> {
    #[account(mut)]
    write_authority: Signer<'info>,

    #[account(
        mut,
        has_one = write_authority,
        close = write_authority,
    )]
    vaa: Account<'info, EncodedVaa>,

    #[account(
        init,
        payer = write_authority,
        space = PostedVaaV1::compute_size(vaa.v1()?.body().payload().as_ref().len()),
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            solana_program::keccak::hash(vaa.v1()?.body().as_ref()).as_ref(),
        ],
        bump,
    )]
    posted_vaa: Account<'info, LegacyAnchorized<4, PostedVaaV1>>,

    system_program: Program<'info, System>,
}

impl<'info> PostVaaV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // We can only create a legacy VAA account if the VAA account was verified.
        //
        // NOTE: We are keeping this here as a fail safe. We should never reach this point because
        // if the VAA is unverified, its version will not be set (so the account context zero-copy
        // getters before this access control will fail).
        require!(
            ctx.accounts.vaa.status == ProcessingStatus::Verified,
            CoreBridgeError::UnverifiedVaa
        );

        // Done.
        Ok(())
    }
}

/// This directive acts as a placeholder in case we want to expand how VAAs are posted.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum PostVaaV1Directive {
    TryOnce,
}

#[access_control(PostVaaV1::constraints(&ctx))]
pub fn post_vaa_v1(ctx: Context<PostVaaV1>, directive: PostVaaV1Directive) -> Result<()> {
    match directive {
        PostVaaV1Directive::TryOnce => try_once(ctx),
    }
}

fn try_once(ctx: Context<PostVaaV1>) -> Result<()> {
    msg!("Directive: TryOnce");

    let encoded_vaa = &ctx.accounts.vaa;

    // This is safe because the VAA integrity has already been verified.
    let vaa = Vaa::parse(&encoded_vaa.buf).unwrap();

    // Verify version.
    require_eq!(
        vaa.version(),
        u8::from(VaaVersion::V1),
        CoreBridgeError::InvalidVaaVersion
    );

    let body = vaa.body();

    ctx.accounts.posted_vaa.set_inner(
        PostedVaaV1 {
            info: PostedVaaV1Info {
                consistency_level: body.consistency_level(),
                timestamp: body.timestamp().into(),
                signature_set: Default::default(),
                guardian_set_index: vaa.guardian_set_index(),
                nonce: body.nonce(),
                sequence: body.sequence(),
                emitter_chain: body.emitter_chain(),
                emitter_address: body.emitter_address(),
            },
            payload: body.payload().as_ref().to_vec(),
        }
        .into(),
    );

    // Done.
    Ok(())
}
