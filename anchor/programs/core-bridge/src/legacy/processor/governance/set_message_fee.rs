use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{BridgeProgramData, Claim, PartialPostedVaaV1, VaaV1MessageHash},
    utils::GOVERNANCE_DECREE_START,
};
use anchor_lang::prelude::*;
use wormhole_io::Readable;
use wormhole_solana_common::SeedPrefix;

const ACTION_SET_MESSAGE_FEE: u8 = 3;

#[derive(Accounts)]
pub struct SetMessageFee<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
    )]
    bridge: Account<'info, BridgeProgramData>,

    #[account(
        seeds = [
            PartialPostedVaaV1::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PartialPostedVaaV1>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes()
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    system_program: Program<'info, System>,
}

impl<'info> SetMessageFee<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let action = crate::utils::require_valid_governance_posted_vaa(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.bridge,
        )?;

        require_eq!(
            action,
            ACTION_SET_MESSAGE_FEE,
            CoreBridgeError::InvalidGovernanceAction
        );

        let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
        let mut data = &acc_info.data.borrow()[GOVERNANCE_DECREE_START..];

        require!(
            <[u8; 24]>::read(&mut data)? == [0; 24],
            CoreBridgeError::U64Overflow
        );

        // Done.
        Ok(())
    }
}

#[access_control(SetMessageFee::accounts(&ctx))]
pub fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let mut data = &acc_info.data.borrow()[(GOVERNANCE_DECREE_START + 24)..];

    ctx.accounts.bridge.fee_lamports = u64::read(&mut data)?;

    // Done.
    Ok(())
}
