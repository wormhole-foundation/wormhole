use crate::{
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, RegisteredEmitter},
    utils::PostedGovernanceVaaV1,
};
use anchor_lang::prelude::*;
use core_bridge_program::state::VaaV1MessageHash;
use wormhole_solana_common::{utils, SeedPrefix};
use wormhole_vaas::payloads::gov::token_bridge::Decree;

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
        seeds = [&try_registered_emitter_seed(&posted_vaa.payload.decree)?],
        bump
    )]
    registered_emitter: Box<Account<'info, RegisteredEmitter>>,

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

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> RegisterChain<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let decree = crate::utils::require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa)?;

        if let Decree::RegisterChain(inner) = decree {
            require!(
                !utils::is_nonzero_slice(inner.foreign_emitter.as_ref()),
                TokenBridgeError::EmitterZeroAddress
            );

            // Done.
            Ok(())
        } else {
            err!(TokenBridgeError::InvalidGovernanceAction)
        }
    }
}

#[access_control(RegisterChain::accounts(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    if let Decree::RegisterChain(inner) = &ctx.accounts.posted_vaa.payload.decree {
        // Set account data for new foreign Token Bridge.
        ctx.accounts
            .registered_emitter
            .set_inner(RegisteredEmitter {
                chain: inner.foreign_chain,
                contract: inner.foreign_emitter.0,
            });
    } else {
        unreachable!()
    }

    // Done.
    Ok(())
}

fn try_registered_emitter_seed(decree: &Decree) -> Result<[u8; 2]> {
    match decree {
        Decree::RegisterChain(inner) => Ok(inner.foreign_chain.to_be_bytes()),
        _ => err!(TokenBridgeError::InvalidGovernanceAction),
    }
}
