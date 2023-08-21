use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, Config, FeeCollector, PartialPostedVaaV1, VaaV1Account},
};
use anchor_lang::{
    prelude::*,
    system_program::{self, Transfer},
};
use ruint::aliases::U256;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct TransferFees<'info> {
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

    #[account(
        mut,
        seeds = [FeeCollector::SEED_PREFIX],
        bump,
    )]
    fee_collector: Account<'info, FeeCollector>,

    /// CHECK: This recipient account must equal the one encoded in the governance VAA.
    #[account(mut)]
    recipient: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> TransferFees<'info> {
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

        match gov_payload.decree().transfer_fees() {
            Some(decree) => {
                require_eq!(
                    decree.chain(),
                    SOLANA_CHAIN,
                    CoreBridgeError::GovernanceForAnotherChain
                );

                let amount = U256::from_be_bytes(decree.amount());
                require_gte!(U256::from(u64::MAX), amount, CoreBridgeError::U64Overflow);

                require_keys_eq!(
                    ctx.accounts.recipient.key(),
                    Pubkey::from(decree.recipient()),
                    CoreBridgeError::InvalidFeeRecipient
                );

                let fee_collector: &AccountInfo = ctx.accounts.fee_collector.as_ref();

                // We cannot remove more than what is required to be rent exempt. We prefer to abort
                // here rather than abort when we attempt the transfer (since the transfer will fail if
                // the lamports in the fee collector account drops below being rent exempt).
                let required_rent =
                    Rent::get().map(|rent| rent.minimum_balance(fee_collector.data_len()))?;
                require_gte!(
                    fee_collector
                        .lamports()
                        .saturating_sub(to_u64_unchecked(&amount)),
                    required_rent,
                    CoreBridgeError::NotEnoughLamports
                );

                // Done.
                Ok(())
            }
            None => err!(CoreBridgeError::InvalidGovernanceAction),
        }
    }
}

#[access_control(TransferFees::constraints(&ctx))]
pub fn transfer_fees(ctx: Context<TransferFees>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let acc_data = acc_info.data.borrow();

    let amount = U256::from_be_bytes(
        super::parse_gov_payload(&acc_data)
            .unwrap()
            .decree()
            .transfer_fees()
            .unwrap()
            .amount(),
    );

    let fee_collector: &AccountInfo = ctx.accounts.fee_collector.as_ref();

    // Finally transfer collected fees to recipient.
    //
    // NOTE: This transfer will not allow us to remove more than what is
    // required to be rent exempt.
    system_program::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.system_program.to_account_info(),
            Transfer {
                from: fee_collector.to_account_info(),
                to: ctx.accounts.recipient.to_account_info(),
            },
            &[&[FeeCollector::SEED_PREFIX, &[ctx.bumps["fee_collector"]]]],
        ),
        to_u64_unchecked(&amount),
    )?;

    // Set the config program data to reflect removing collected fees.
    ctx.accounts.config.last_lamports = fee_collector.lamports();

    // Done.
    Ok(())
}

fn to_u64_unchecked(value: &U256) -> u64 {
    value.as_limbs()[0]
}
