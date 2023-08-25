use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct SecureRegisteredEmitter<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [legacy_registered_emitter.chain.to_be_bytes().as_ref()],
        bump,
    )]
    registered_emitter: Account<'info, RegisteredEmitter>,

    /// This account should be created using only the emitter chain ID as its seed. Instead, it uses
    /// both emitter chain and address to derive this PDA address. Having both of these as seeds
    /// potentially allows for multiple emitters to be registered for a given chain ID (when there
    /// should only be one).
    ///
    /// See the new `register_chain` instruction handler for the correct way to create this account.
    #[account(
        seeds = [
            legacy_registered_emitter.chain.to_be_bytes().as_ref(),
            legacy_registered_emitter.contract.as_ref(),
        ],
        bump,
    )]
    legacy_registered_emitter: Account<'info, RegisteredEmitter>,

    system_program: Program<'info, System>,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum SecureRegisteredEmitterDirective {
    Init,
    CloseLegacy,
}

pub fn secure_registered_emitter(
    ctx: Context<SecureRegisteredEmitter>,
    directive: SecureRegisteredEmitterDirective,
) -> Result<()> {
    match directive {
        SecureRegisteredEmitterDirective::Init => {
            msg!("Directive: Init");
            init(ctx)
        }
        SecureRegisteredEmitterDirective::CloseLegacy => {
            msg!("Directive: CloseLegacy");
            close_legacy(ctx)
        }
    }
}

fn init(ctx: Context<SecureRegisteredEmitter>) -> Result<()> {
    let registered = &mut ctx.accounts.registered_emitter;
    require_eq!(
        registered.chain,
        0,
        TokenBridgeError::EmitterAlreadyRegistered
    );

    // Copy registered emitter account.
    ctx.accounts
        .registered_emitter
        .set_inner(*ctx.accounts.legacy_registered_emitter.as_ref());

    // Done.
    Ok(())
}

fn close_legacy(ctx: Context<SecureRegisteredEmitter>) -> Result<()> {
    require_eq!(
        ctx.accounts.legacy_registered_emitter.chain,
        ctx.accounts.registered_emitter.chain,
        TokenBridgeError::RegisteredEmitterMismatch
    );

    err!(TokenBridgeError::UnsupportedInstructionDirective)

    //ctx.accounts.legacy_registered_emitter.close(ctx.accounts.payer.to_account_info())
}
