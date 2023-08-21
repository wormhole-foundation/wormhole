use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, Config, PartialPostedVaaV1, VaaV1Account},
};
use anchor_lang::prelude::*;
use ruint::aliases::U256;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct SetMessageFee<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, Config>,

    #[account(
        seeds = [
            PartialPostedVaaV1::SEED_PREFIX,
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
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = &ctx.accounts.posted_vaa;

        let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
        let acc_data = acc_info.try_borrow_data()?;

        let gov_payload = super::require_valid_governance_posted_vaa(
            vaa.details(),
            &acc_data,
            vaa.guardian_set_index,
            &ctx.accounts.config,
        )?;

        match gov_payload.decree().set_message_fee() {
            Some(decree) => {
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
            None => err!(CoreBridgeError::InvalidGovernanceAction),
        }
    }
}

#[access_control(SetMessageFee::constraints(&ctx))]
pub fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let acc_data = acc_info.data.borrow();

    let fee = U256::from_be_bytes(
        super::parse_gov_payload(&acc_data)
            .unwrap()
            .decree()
            .set_message_fee()
            .unwrap()
            .fee(),
    );

    ctx.accounts.config.fee_lamports = fee.as_limbs()[0];

    // Done.
    Ok(())
}
