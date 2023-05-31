use std::io;

use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    message::{
        require_valid_governance_posted_vaa, CoreBridgeGovernance, PostedGovernanceVaaV1,
        WormDecode, WormEncode,
    },
    state::{BridgeProgramData, Claim, FeeCollector, VaaV1LegacyAccount},
};
use anchor_lang::{
    prelude::*,
    system_program::{self, Transfer},
};
use wormhole_common::{utils, SeedPrefix};

#[derive(Debug, Clone)]
struct TransferFeesDecree {
    zeros: [u8; 24],
    amount: u64,
    recipient: Pubkey,
}

impl WormDecode for TransferFeesDecree {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let zeros = <[u8; 24]>::decode_reader(reader)?;
        let amount = u64::decode_reader(reader)?;
        let recipient = Pubkey::decode_reader(reader)?;
        Ok(Self {
            zeros,
            amount,
            recipient,
        })
    }
}

impl WormEncode for TransferFeesDecree {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.zeros.encode(writer)?;
        self.amount.encode(writer)?;
        self.recipient.encode(writer)
    }
}

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
            PostedGovernanceVaaV1::<TransferFeesDecree>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1<TransferFeesDecree>>,

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
        let msg =
            require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa, &ctx.accounts.bridge)?;

        // We expect a specific governance header.
        require!(
            msg.header == CoreBridgeGovernance::TransferFees.try_into()?,
            CoreBridgeError::InvalidGovernanceAction
        );

        require!(
            !utils::is_nonzero_array(&msg.decree.zeros),
            CoreBridgeError::U64Overflow
        );

        // Done.
        Ok(())
    }
}

#[access_control(TransferFees::accounts(&ctx))]
pub fn transfer_fees(ctx: Context<TransferFees>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let decree = &ctx.accounts.posted_vaa.payload.decree;

    // Finally read the encoded fee and set it in the bridge program data.
    let lamports = decree.amount;

    let fee_collector = &ctx.accounts.fee_collector;
    let last_lamports = {
        let acct_info = fee_collector.to_account_info();

        // We cannot remove more than what is required to be rent exempt. We prefer to abort
        // here rather than abort when we attempt the transfer (since the transfer will fail if
        // the lamports in the fee collector account drops below being rent exempt).
        let required_rent = Rent::get().map(|rent| rent.minimum_balance(acct_info.data_len()))?;
        let remaining = acct_info.lamports().saturating_sub(lamports);
        require_gte!(remaining, required_rent, CoreBridgeError::NotEnoughLamports);

        remaining
    };

    // Set the bridge program data to reflect removing collected fees.
    ctx.accounts.bridge.last_lamports = last_lamports;

    // The encoded recipient must equal the recipient account passed in.
    let recipient = &ctx.accounts.recipient;
    require_keys_eq!(recipient.key(), decree.recipient);

    // Finally transfer collected fees to recipient.
    //
    // NOTE: This transfer will not allow us to remove more than what is
    // required to be rent exempt.
    system_program::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.system_program.to_account_info(),
            Transfer {
                from: fee_collector.to_account_info(),
                to: recipient.to_account_info(),
            },
            &[&[FeeCollector::seed_prefix(), &[ctx.bumps["fee_collector"]]]],
        ),
        lamports,
    )
}
