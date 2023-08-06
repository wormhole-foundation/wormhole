use crate::{
    error::TokenBridgeError,
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    legacy::utils::LegacyAnchorized, sdk::cpi::CoreBridge, zero_copy::EncodedVaa,
};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        mut,
        owner = core_bridge_program::ID
    )]
    vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            EncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().emitter_address().as_ref(),
            &EncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().emitter_chain().to_be_bytes(),
            &EncodedVaa::try_v1(&vaa.try_borrow_data()?)?.body().sequence().to_be_bytes(),
        ],
        bump,
    )]
    claim: Account<'info, LegacyAnchorized<0, Claim>>,

    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [try_decree_foreign_chain(&vaa.try_borrow_data()?)?.to_be_bytes().as_ref()],
        bump,
    )]
    registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

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
            try_decree_foreign_chain(&vaa.try_borrow_data()?)?.to_be_bytes().as_ref(),
            try_decree_foreign_emitter(&vaa.try_borrow_data()?)?.as_ref(),
        ],
        bump,
    )]
    legacy_registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        super::require_valid_governance_encoded_vaa(&ctx.accounts.vaa.data.borrow()).map(|_| ())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>) -> Result<()> {
    // Mark the claim as complete. The account only exists to ensure that the VAA is not processed,
    // so this value does not matter. But the legacy program set this data to true.
    ctx.accounts.claim.is_complete = true;

    // Deserialize and set data in registered emitter accounts.
    {
        let acc_data = ctx.accounts.vaa.data.borrow();
        let encoded_vaa = EncodedVaa::parse(&acc_data).unwrap();
        let gov_payload = TokenBridgeGovPayload::try_from(encoded_vaa.v1().unwrap().payload())
            .unwrap()
            .decree();
        let decree = gov_payload.register_chain().unwrap();

        let registered = RegisteredEmitter {
            chain: decree.foreign_chain(),
            contract: decree.foreign_emitter(),
        };

        ctx.accounts.registered_emitter.set_inner(registered.into());
        ctx.accounts
            .legacy_registered_emitter
            .set_inner(registered.into());
    }

    // Determine if we can close the vaa account.
    let payer = &ctx.accounts.payer;
    let write_authority = EncodedVaa::parse(&ctx.accounts.vaa.data.borrow())
        .unwrap()
        .write_authority();
    if payer.key() == write_authority {
        core_bridge_program::cpi::process_encoded_vaa(
            CpiContext::new(
                ctx.accounts.core_bridge_program.to_account_info(),
                core_bridge_program::cpi::accounts::ProcessEncodedVaa {
                    write_authority: payer.to_account_info(),
                    encoded_vaa: ctx.accounts.vaa.to_account_info(),
                    guardian_set: None,
                },
            ),
            core_bridge_program::ProcessEncodedVaaDirective::CloseVaaAccount,
        )
    } else {
        Ok(())
    }
}

fn try_decree_foreign_chain(vaa_acc_data: &[u8]) -> Result<u16> {
    let vaa = EncodedVaa::try_v1(vaa_acc_data)?;
    let gov_payload = TokenBridgeGovPayload::try_from(vaa.body().payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;
    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_chain())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}

fn try_decree_foreign_emitter(vaa_acc_data: &[u8]) -> Result<[u8; 32]> {
    let vaa = EncodedVaa::try_v1(vaa_acc_data)?;
    let gov_payload = TokenBridgeGovPayload::try_from(vaa.body().payload())
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;
    gov_payload
        .decree()
        .register_chain()
        .map(|decree| decree.foreign_emitter())
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}
