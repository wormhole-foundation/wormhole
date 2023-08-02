use crate::{
    error::CoreBridgeError,
    state::{EncodedVaa, PostedVaaV1Bytes, PostedVaaV1Metadata, ProcessingStatus},
    types::VaaVersion,
};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::Vaa;
use wormhole_solana_common::{NewAccountSize, SeedPrefix};

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
        space = PostedVaaV1Bytes::compute_size(vaa.payload_size()?),
        //seeds = [PostedVaaV1Bytes::seed_prefix(), vaa.compute_message_hash()?.as_ref()],
        seeds = [PostedVaaV1Bytes::seed_prefix(), &Vaa::parse(&mut &vaa.bytes).unwrap().body().digest()],
        bump,
    )]
    posted_vaa: Account<'info, PostedVaaV1Bytes>,

    system_program: Program<'info, System>,
}

impl<'info> PostVaaV1<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // We can only create a legacy VAA account if the VAA account was verified.
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

#[access_control(PostVaaV1::accounts(&ctx))]
pub fn post_vaa_v1(ctx: Context<PostVaaV1>, directive: PostVaaV1Directive) -> Result<()> {
    match directive {
        PostVaaV1Directive::TryOnce => {
            msg!("Directive: TryOnce");
            let encoded_vaa = &ctx.accounts.vaa;
            let mut vaa_buf: &[u8] = &encoded_vaa.bytes;

            // This is safe because the VAA integrity has already been verified.
            let vaa = Vaa::parse(&mut vaa_buf).unwrap();

            // Verify version.
            require_eq!(
                vaa.version(),
                u8::from(VaaVersion::V1),
                CoreBridgeError::InvalidVaaVersion
            );

            let body = vaa.body();

            ctx.accounts.posted_vaa.set_inner(PostedVaaV1Bytes {
                meta: PostedVaaV1Metadata {
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
            });

            // Done.
            Ok(())
        }
    }
}
