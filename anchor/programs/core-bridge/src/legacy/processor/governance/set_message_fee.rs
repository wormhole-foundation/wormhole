use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{BridgeProgramData, Claim, VaaV1LegacyAccount},
    utils::PostedGovernanceVaaV1,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::SeedPrefix;
use wormhole_vaas::{payloads::gov::core_bridge::Decree, U256};

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
            PostedGovernanceVaaV1::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1>,

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
        let decree = crate::utils::require_valid_governance_posted_vaa(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.bridge,
        )?;

        if let Decree::SetMessageFee(inner) = decree {
            require_gte!(
                U256::from(u64::MAX),
                inner.fee,
                CoreBridgeError::U64Overflow
            );

            // Done.
            Ok(())
        } else {
            err!(CoreBridgeError::InvalidGovernanceAction)
        }
    }
}

#[access_control(SetMessageFee::accounts(&ctx))]
pub fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // We know this is the only variant that can be present given access control.
    if let Decree::SetMessageFee(inner) = &ctx.accounts.posted_vaa.payload.decree {
        ctx.accounts.bridge.fee_lamports = inner.fee.try_into().unwrap();
    } else {
        unreachable!()
    }

    // Done.
    Ok(())
}
