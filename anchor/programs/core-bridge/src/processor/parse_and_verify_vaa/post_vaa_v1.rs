use crate::{
    error::CoreBridgeError,
    message::{VaaV1MessageInfo, WormDecode},
    state::{EncodedVaa, PostedVaaV1Bytes, PostedVaaV1Metadata, ProcessingStatus},
    types::VaaVersion,
};
use anchor_lang::prelude::*;
use wormhole_common::{NewAccountSize, SeedPrefix};

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
        seeds = [PostedVaaV1Bytes::seed_prefix(), vaa.compute_message_hash()?.as_ref()],
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

            // Verify version.
            let version = encoded_vaa.version;
            require!(
                version == VaaVersion::V1,
                CoreBridgeError::InvalidVaaVersion
            );

            // First byte is the version, which is already verified.
            vaa_buf = &vaa_buf[1..];

            // Deserialize guardian set index and number of signatures (so we can skip to the
            // message body).
            let guardian_set_index = u32::decode(&mut vaa_buf)?;
            let num_signatures = u8::decode(&mut vaa_buf).map(usize::from)?;

            vaa_buf = &vaa_buf[(66 * num_signatures)..];

            let VaaV1MessageInfo {
                timestamp,
                nonce,
                emitter_chain,
                emitter_address,
                sequence,
                finality,
            } = VaaV1MessageInfo::decode(&mut vaa_buf)?;

            let mut payload = Vec::with_capacity(vaa_buf.len());
            payload.extend_from_slice(vaa_buf);

            // let guardian_set_index = vaa.guardian_set_index;
            ctx.accounts.posted_vaa.set_inner(PostedVaaV1Bytes {
                meta: PostedVaaV1Metadata {
                    finality,
                    timestamp,
                    signature_set: Default::default(),
                    guardian_set_index,
                    nonce,
                    sequence,
                    emitter_chain,
                    emitter_address,
                },
                payload,
            });

            // Done.
            Ok(())
        }
    }
}
