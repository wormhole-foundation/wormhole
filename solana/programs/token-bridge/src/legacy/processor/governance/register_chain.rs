use crate::{
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use core_bridge_program::{zero_copy::PostedVaaV1, CoreBridge};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// This account should be created using only the emitter chain ID as its seed. Instead, it uses
    /// both emitter chain and address to derive this PDA address. Having both of these as seeds
    /// potentially allows for multiple emitters to be registered for a given chain ID (when there
    /// should only be one).
    ///
    /// See the new `register_chain` instruction handler for the correct way to create this account.
    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [
            try_decree_foreign_chain(&posted_vaa.try_borrow_data()?)?.to_be_bytes().as_ref(),
            try_decree_foreign_emitter(&posted_vaa.try_borrow_data()?)?.as_ref(),
        ],
        bump,
    )]
    registered_emitter: Account<'info, RegisteredEmitter>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump,
        seeds::program = core_bridge_program,
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
    claim: Account<'info, Claim>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = &ctx.accounts.posted_vaa;
        super::require_valid_posted_governance_vaa(&vaa.key(), &vaa.data.borrow()).map(|_| ())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();
    let gov_payload = TokenBridgeGovPayload::parse(vaa.payload())
        .unwrap()
        .decree();
    let register_chain = gov_payload.register_chain().unwrap();

    ctx.accounts
        .registered_emitter
        .set_inner(RegisteredEmitter {
            chain: register_chain.foreign_chain(),
            contract: register_chain.foreign_emitter(),
        });

    // Done.
    Ok(())
}

fn try_decree_foreign_chain(vaa_acc_data: &[u8]) -> Result<u16> {
    let vaa = PostedVaaV1::parse(vaa_acc_data)?;
    let gov_payload = TokenBridgeGovPayload::parse(vaa.payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;

    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_chain())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}

fn try_decree_foreign_emitter(vaa_acc_data: &[u8]) -> Result<[u8; 32]> {
    let vaa = PostedVaaV1::parse(vaa_acc_data)?;
    let gov_payload = TokenBridgeGovPayload::parse(vaa.payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;

    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_emitter())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}
