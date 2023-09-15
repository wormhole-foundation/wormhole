use crate::state::RegisteredEmitter;
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct SecureRegisteredEmitter<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init,
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

pub fn secure_registered_emitter(ctx: Context<SecureRegisteredEmitter>) -> Result<()> {
    let registered = &ctx.accounts.legacy_registered_emitter;

    ctx.accounts
        .registered_emitter
        .set_inner(RegisteredEmitter {
            chain: registered.chain,
            contract: registered.contract,
        });

    // Done.
    Ok(())
}
