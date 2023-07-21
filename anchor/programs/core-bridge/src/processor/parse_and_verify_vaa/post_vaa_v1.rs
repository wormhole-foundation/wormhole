use crate::{
    error::CoreBridgeError,
    state::{EncodedVaa, PostedVaaV1Bytes, PostedVaaV1Metadata, ProcessingStatus},
    types::VaaVersion,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{NewAccountSize, SeedPrefix};
use wormhole_vaas::{PayloadKind, Readable, VaaBody};

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
            let guardian_set_index = u32::read(&mut vaa_buf)?;
            let num_signatures = u8::read(&mut vaa_buf).map(usize::from)?;

            vaa_buf = &vaa_buf[(66 * num_signatures)..];

            let VaaBody {
                timestamp,
                nonce,
                emitter_chain,
                emitter_address,
                sequence,
                consistency_level,
                payload,
            } = VaaBody::read(&mut vaa_buf)?;

            // When VaaBody is deserialized, there is only one variant: Binary.
            if let PayloadKind::Binary(payload) = payload {
                ctx.accounts.posted_vaa.set_inner(PostedVaaV1Bytes {
                    meta: PostedVaaV1Metadata {
                        consistency_level,
                        timestamp: timestamp.into(),
                        signature_set: Default::default(),
                        guardian_set_index,
                        nonce,
                        sequence: u64::try_from(sequence).unwrap(),
                        emitter_chain,
                        emitter_address: emitter_address.0,
                    },
                    payload,
                });
            } else {
                unreachable!()
            }

            // Done.
            Ok(())
        }
    }
}
