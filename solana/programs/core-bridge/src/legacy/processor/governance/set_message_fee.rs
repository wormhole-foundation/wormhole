use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    legacy::{instruction::EmptyArgs, utils::LegacyAccount},
    state::{Claim, Config},
    zero_copy::PostedVaaV1,
};
use anchor_lang::prelude::*;
use ruint::aliases::U256;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;

#[derive(Accounts)]
pub struct SetMessageFee<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAccount<0, Config>>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump
    )]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_address().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_chain().to_be_bytes().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.sequence().to_be_bytes().as_ref(),
        ],
        bump,
    )]
    claim: Account<'info, LegacyAccount<0, Claim>>,

    system_program: Program<'info, System>,
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for SetMessageFee<'info>
{
    const LOG_IX_NAME: &'static str = "LegacySetMessageFee";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = set_message_fee;
}

impl<'info> SetMessageFee<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let acc_data = ctx.accounts.posted_vaa.try_borrow_data()?;
        let gov_payload =
            super::require_valid_posted_governance_vaa(&acc_data, &ctx.accounts.config)?;

        let decree = gov_payload
            .set_message_fee()
            .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))?;

        require_eq!(
            decree.chain(),
            SOLANA_CHAIN,
            CoreBridgeError::GovernanceForAnotherChain
        );

        let fee = U256::from_be_bytes(decree.fee());
        require_gte!(U256::from(u64::MAX), fee, CoreBridgeError::U64Overflow);

        // Done.
        Ok(())
    }
}

#[access_control(SetMessageFee::constraints(&ctx))]
fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();
    let gov_payload = CoreBridgeGovPayload::parse(vaa.payload()).unwrap().decree();

    // Uint encodes limbs in little endian, so we will take the first u64 value.
    let fee = U256::from_be_bytes(gov_payload.set_message_fee().unwrap().fee());
    ctx.accounts.config.fee_lamports = fee.as_limbs()[0];

    // Done.
    Ok(())
}
