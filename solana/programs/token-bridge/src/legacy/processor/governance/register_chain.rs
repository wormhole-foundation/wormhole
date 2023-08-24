use crate::{
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    state::{PartialPostedVaaV1, VaaV1Account},
    CoreBridge,
};
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
            try_new_foreign_chain(posted_vaa.as_ref())?.as_ref(),
            try_new_foreign_emitter(posted_vaa.as_ref())?.as_ref(),
        ],
        bump,
    )]
    registered_emitter: Account<'info, RegisteredEmitter>,

    #[account(
        seeds = [
            PartialPostedVaaV1::SEED_PREFIX,
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump,
        seeds::program = core_bridge_program,
    )]
    posted_vaa: Account<'info, PartialPostedVaaV1>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes(),
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
        let acc_info: &AccountInfo = vaa.as_ref();
        super::require_valid_governance_posted_vaa(vaa.details(), &acc_info.data.borrow())
            .map(|_| ())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let acc_data = acc_info.data.borrow();

    let gov_payload = super::parse_acc_data(&acc_data).unwrap().decree();
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

fn try_new_foreign_chain(acc_info: &AccountInfo) -> Result<[u8; 2]> {
    let acc_data = &acc_info.try_borrow_data()?;
    let gov_payload = super::parse_acc_data(acc_data)?;
    match gov_payload.decree().register_chain() {
        Some(decree) => Ok(decree.foreign_chain().to_be_bytes()),
        None => err!(TokenBridgeError::InvalidGovernanceAction),
    }
}

fn try_new_foreign_emitter(acc_info: &AccountInfo) -> Result<[u8; 32]> {
    let acc_data = &acc_info.try_borrow_data()?;
    let gov_payload = super::parse_acc_data(acc_data)?;
    match gov_payload.decree().register_chain() {
        Some(decree) => Ok(decree.foreign_emitter()),
        None => err!(TokenBridgeError::InvalidGovernanceAction),
    }
}
