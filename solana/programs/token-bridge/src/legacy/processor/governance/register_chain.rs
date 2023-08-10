use crate::{
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, RegisteredEmitter},
    utils::GOVERNANCE_DECREE_START,
};
use anchor_lang::prelude::*;
use core_bridge_program::state::{PartialPostedVaaV1, VaaV1MessageHash};
use wormhole_raw_vaas::token_bridge::gov;
use wormhole_solana_common::SeedPrefix;

const ACTION_REGISTER_CHAIN: u8 = 1;

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
        bump
    )]
    registered_emitter: Account<'info, RegisteredEmitter>,

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

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let action = crate::utils::require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa)?;

        require_eq!(
            action,
            ACTION_REGISTER_CHAIN,
            TokenBridgeError::InvalidGovernanceAction
        );

        // Done.
        Ok(())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let data = &acc_info.data.borrow()[GOVERNANCE_DECREE_START..];
    let decree = gov::RegisterChain::parse(data).unwrap();

    ctx.accounts
        .registered_emitter
        .set_inner(RegisteredEmitter {
            chain: decree.foreign_chain(),
            contract: decree.foreign_emitter(),
        });

    // Done.
    Ok(())
}

fn try_new_foreign_chain(acc_info: &AccountInfo) -> Result<[u8; 2]> {
    let data = &acc_info.try_borrow_data()?[GOVERNANCE_DECREE_START..];
    match gov::RegisterChain::parse(data) {
        Ok(decree) => Ok(decree.foreign_chain().to_be_bytes()),
        Err(_) => err!(TokenBridgeError::InvalidGovernanceAction),
    }
}

fn try_new_foreign_emitter(acc_info: &AccountInfo) -> Result<[u8; 32]> {
    let data = &acc_info.try_borrow_data()?[GOVERNANCE_DECREE_START..];
    match gov::RegisterChain::parse(data) {
        Ok(decree) => Ok(decree.foreign_emitter()),
        Err(_) => err!(TokenBridgeError::InvalidGovernanceAction),
    }
}
