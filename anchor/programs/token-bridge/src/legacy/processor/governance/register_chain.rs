use std::io;

use crate::{
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    message::{require_valid_governance_posted_vaa, TokenBridgeGovernance},
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    message::PostedGovernanceVaaV1,
    state::VaaV1LegacyAccount,
    types::{ChainId, ExternalAddress},
    WormDecode, WormEncode,
};
use wormhole_common::{utils, SeedPrefix};

#[derive(Debug, Clone)]
struct RegisterChainDecree {
    chain: ChainId,
    emitter: ExternalAddress,
}

impl WormDecode for RegisterChainDecree {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let chain = ChainId::decode_reader(reader)?;
        let emitter = ExternalAddress::decode_reader(reader)?;
        Ok(Self { chain, emitter })
    }
}

impl WormEncode for RegisterChainDecree {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.chain.encode(writer)?;
        self.emitter.encode(writer)
    }
}

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// TODO: Write note about legacy vs now.
    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [&posted_vaa.payload.decree.chain.to_be_bytes()],
        bump
    )]
    registered_emitter: Box<Account<'info, RegisteredEmitter>>,

    #[account(
        seeds = [
            PostedGovernanceVaaV1::<RegisterChainDecree>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1<RegisterChainDecree>>,

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

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> RegisterChain<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let msg = require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa)?;

        // We expect a specific governance header.
        require!(
            msg.header == TokenBridgeGovernance::RegisterChain.try_into()?,
            TokenBridgeError::InvalidGovernanceAction
        );

        require!(
            !utils::is_nonzero_slice(msg.decree.emitter.as_ref()),
            TokenBridgeError::EmitterZeroAddress
        );

        // Done.
        Ok(())
    }
}

#[access_control(RegisterChain::accounts(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // Set account data for new foreign Token Bridge.
    ctx.accounts
        .registered_emitter
        .set_inner(RegisteredEmitter {
            chain: ctx.accounts.posted_vaa.payload.decree.chain,
            contract: ctx.accounts.posted_vaa.payload.decree.emitter,
        });

    // Done.
    Ok(())
}
