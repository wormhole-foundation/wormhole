use crate::{
    error::TokenBridgeError,
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use core_bridge_program::state::ZeroCopyEncodedVaa;
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [try_decree_foreign_chain(&vaa)?.to_be_bytes().as_ref()],
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
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [
            try_decree_foreign_chain(&vaa)?.to_be_bytes().as_ref(),
            try_decree_foreign_emitter(&vaa)?.as_ref(),
        ],
        bump,
    )]
    legacy_registered_emitter: Account<'info, RegisteredEmitter>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(owner = core_bridge_program::ID)]
    vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            ZeroCopyEncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().emitter_address().as_ref(),
            &ZeroCopyEncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().emitter_chain().to_be_bytes(),
            &ZeroCopyEncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().sequence().to_be_bytes(),
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    system_program: Program<'info, System>,
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        super::require_valid_governance_encoded_vaa(&ctx.accounts.vaa.data.borrow()).map(|_| ())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.vaa.data.borrow();

    let vaa = ZeroCopyEncodedVaa::parse(&acc_data).unwrap().v1().unwrap();
    let gov_payload = TokenBridgeGovPayload::try_from(vaa.payload())
        .unwrap()
        .decree();
    let decree = gov_payload.register_chain().unwrap();

    ctx.accounts
        .registered_emitter
        .set_inner(RegisteredEmitter {
            chain: decree.foreign_chain(),
            contract: decree.foreign_emitter(),
        });

    // Done.
    Ok(())
}

fn try_decree_foreign_chain(acc_info: &AccountInfo) -> Result<u16> {
    let acc_data = &acc_info.try_borrow_data()?;
    let gov_payload = super::parse_acc_data(acc_data)?;
    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_chain())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}

fn try_decree_foreign_emitter(acc_info: &AccountInfo) -> Result<[u8; 32]> {
    let acc_data = acc_info.try_borrow_data()?;
    let gov_payload = super::parse_acc_data(&acc_data)?;
    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_emitter())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}
