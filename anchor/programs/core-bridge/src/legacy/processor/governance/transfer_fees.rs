use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{BridgeProgramData, Claim, FeeCollector, PartialPostedVaaV1, VaaV1MessageHash},
    utils::GOVERNANCE_DECREE_START,
};
use anchor_lang::{
    prelude::*,
    system_program::{self, Transfer},
};
use wormhole_io::Readable;
use wormhole_solana_common::SeedPrefix;

const ACTION_TRANSFER_FEES: u8 = 4;

#[derive(Accounts)]
pub struct TransferFees<'info> {
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

    #[account(
        mut,
        seeds = [FeeCollector::seed_prefix()],
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
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let action = crate::utils::require_valid_governance_posted_vaa(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.bridge,
        )?;

        require_eq!(
            action,
            ACTION_TRANSFER_FEES,
            CoreBridgeError::InvalidGovernanceAction
        );

        let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
        let mut data = &acc_info.data.borrow()[GOVERNANCE_DECREE_START..];

        require!(
            <[u8; 24]>::read(&mut data).unwrap() == [0; 24],
            CoreBridgeError::U64Overflow
        );
        let lamports = u64::read(&mut data).unwrap();

        let recipient = &ctx.accounts.recipient;
        require_keys_eq!(
            recipient.key(),
            Pubkey::new_from_array(Readable::read(&mut data).unwrap()),
            CoreBridgeError::InvalidFeeRecipient
        );

        let fee_collector: &AccountInfo = ctx.accounts.fee_collector.as_ref();

        // We cannot remove more than what is required to be rent exempt. We prefer to abort
        // here rather than abort when we attempt the transfer (since the transfer will fail if
        // the lamports in the fee collector account drops below being rent exempt).
        let required_rent =
            Rent::get().map(|rent| rent.minimum_balance(fee_collector.data_len()))?;
        require_gte!(
            fee_collector.lamports().saturating_sub(lamports),
            required_rent,
            CoreBridgeError::NotEnoughLamports
        );

        // Done.
        Ok(())
    }
}

#[access_control(TransferFees::accounts(&ctx))]
pub fn transfer_fees(ctx: Context<TransferFees>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let mut data = &acc_info.data.borrow()[(GOVERNANCE_DECREE_START + 24)..];
    let amount = u64::read(&mut data).unwrap();

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
            &[&[FeeCollector::seed_prefix(), &[ctx.bumps["fee_collector"]]]],
        ),
        amount,
    )?;

    // Set the bridge program data to reflect removing collected fees.
    ctx.accounts.bridge.last_lamports = fee_collector.lamports();

    // Done.
    Ok(())
}
