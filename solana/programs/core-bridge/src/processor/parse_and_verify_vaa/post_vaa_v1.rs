use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAnchorized,
    state::{PostedVaaV1, PostedVaaV1Info},
    zero_copy::{self, LoadZeroCopy},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct PostVaaV1<'info> {
    #[account(mut)]
    write_authority: Signer<'info>,

    /// CHECK: Encoded VAA, whose body will be serialized into the posted VAA account.
    encoded_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = write_authority,
        space = try_compute_size(&encoded_vaa)?,
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            try_message_hash(&encoded_vaa)?.as_ref(),
        ],
        bump,
    )]
    posted_vaa: Account<'info, LegacyAnchorized<PostedVaaV1>>,

    system_program: Program<'info, System>,
}

pub fn post_vaa_v1(ctx: Context<PostVaaV1>) -> Result<()> {
    // This is safe because we checked that the VAA version in the encoded VAA account is V1.
    let encoded_vaa = zero_copy::EncodedVaa::load(&ctx.accounts.encoded_vaa).unwrap();
    let vaa = encoded_vaa.as_vaa().unwrap();
    let v1 = vaa.v1().unwrap();

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

fn try_compute_size(vaa: &AccountInfo) -> Result<usize> {
    let encoded_vaa = zero_copy::EncodedVaa::load(vaa)?;
    let payload_size = encoded_vaa
        .as_vaa()?
        .v1()
        .ok_or(error!(CoreBridgeError::InvalidVaaVersion))?
        .body()
        .payload()
        .as_ref()
        .len();

    let acc_size = PostedVaaV1::compute_size(payload_size);

    // CPI to create the posted VAA account will fail if the size of the VAA payload is too large.
    require!(
        acc_size <= 10_240,
        CoreBridgeError::PostedVaaPayloadTooLarge
    );

    Ok(acc_size)
}

fn try_message_hash(vaa: &AccountInfo) -> Result<solana_program::keccak::Hash> {
    let encoded_vaa = zero_copy::EncodedVaa::load(vaa)?;
    let vaa = encoded_vaa.as_vaa()?;
    let body = vaa
        .v1()
        .ok_or(error!(CoreBridgeError::InvalidVaaVersion))?
        .body();

    Ok(solana_program::keccak::hash(body.as_ref()))
}
