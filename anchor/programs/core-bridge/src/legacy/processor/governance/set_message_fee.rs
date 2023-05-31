use std::io;

use crate::{
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    message::{
        require_valid_governance_posted_vaa, CoreBridgeGovernance, PostedGovernanceVaaV1,
        WormDecode, WormEncode,
    },
    state::{BridgeProgramData, Claim, VaaV1LegacyAccount},
};
use anchor_lang::prelude::*;
use wormhole_common::{utils, SeedPrefix};

#[derive(Debug, Clone)]
struct SetMessageFeeDecree {
    zeros: [u8; 24],
    new_fee: u64,
}

impl WormDecode for SetMessageFeeDecree {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let zeros = <[u8; 24]>::decode_reader(reader)?;
        let new_fee = u64::decode_reader(reader)?;
        Ok(Self { zeros, new_fee })
    }
}

impl WormEncode for SetMessageFeeDecree {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.zeros.encode(writer)?;
        self.new_fee.encode(writer)
    }
}

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
            PostedGovernanceVaaV1::<SetMessageFeeDecree>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1<SetMessageFeeDecree>>,

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
        let msg =
            require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa, &ctx.accounts.bridge)?;

        // We expect a specific governance header.
        require!(
            msg.header == CoreBridgeGovernance::SetMessageFee.try_into()?,
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

#[access_control(SetMessageFee::accounts(&ctx))]
pub fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // Finally read the encoded fee and set it in the bridge program data.
    ctx.accounts.bridge.fee_lamports = ctx.accounts.posted_vaa.payload.decree.new_fee;

    // Done.
    Ok(())
}
