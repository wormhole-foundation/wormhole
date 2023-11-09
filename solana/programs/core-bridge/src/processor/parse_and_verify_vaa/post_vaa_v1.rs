use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAnchorized,
    state::{EncodedVaa, PostedVaaV1, PostedVaaV1Info, ProcessingStatus},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct PostVaaV1<'info> {
    /// Payer to create the posted VAA account. This instruction allows anyone with an encoded VAA
    /// to create a posted VAA account.
    #[account(mut)]
    payer: Signer<'info>,

    /// Encoded VAA, whose body will be serialized into the posted VAA account.
    ///
    /// NOTE: This instruction handler only exists to support integrators that still rely on posted
    /// VAA accounts. While we encourage integrators to use the encoded VAA account instead, we
    /// allow a pathway to convert the encoded VAA into a posted VAA. However, the payload is
    /// restricted to 9.5KB, which is much larger than what was possible with the old implementation
    /// using the legacy post vaa instruction. The Core Bridge program will not support posting VAAs
    /// larger than this payload size.
    #[account(
        constraint = encoded_vaa.status == ProcessingStatus::Verified @ CoreBridgeError::UnverifiedVaa
    )]
    encoded_vaa: Account<'info, EncodedVaa>,

    #[account(
        init,
        payer = payer,
        space = PostedVaaV1::compute_size(
            encoded_vaa
                .as_vaa()?
                .to_v1()?
                .body()
                .payload()
                .as_ref()
                .len()
        ),
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            solana_program::keccak::hash(
                encoded_vaa
                    .as_vaa()?
                    .to_v1()?
                    .body().as_ref()
            ).as_ref()
        ],
        bump,
    )]
    posted_vaa: Account<'info, LegacyAnchorized<PostedVaaV1>>,

    system_program: Program<'info, System>,
}

impl<'info> PostVaaV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // CPI to create the posted VAA account will fail if the size of the VAA payload is too large.
        require!(
            ctx.accounts.encoded_vaa.buf.len() <= 9_728,
            CoreBridgeError::PostedVaaPayloadTooLarge
        );

        let encoded_vaa = ctx.accounts.encoded_vaa.as_vaa()?;
        encoded_vaa
            .v1()
            .ok_or(error!(CoreBridgeError::InvalidVaaVersion))?;

        // Done.
        Ok(())
    }
}

#[access_control(PostVaaV1::constraints(&ctx))]
pub fn post_vaa_v1(ctx: Context<PostVaaV1>) -> Result<()> {
    // This is safe because we checked that the VAA version in the encoded VAA account is V1.
    let v1 = ctx.accounts.encoded_vaa.as_vaa().unwrap().to_v1().unwrap();
    let body = v1.body();

    ctx.accounts.posted_vaa.set_inner(
        PostedVaaV1 {
            info: PostedVaaV1Info {
                consistency_level: body.consistency_level(),
                timestamp: body.timestamp().into(),
                signature_set: Default::default(),
                guardian_set_index: v1.guardian_set_index(),
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
