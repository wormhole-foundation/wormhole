use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// VAA account, which may either be the new EncodedVaa account or legacy PostedVaaV1
    /// account.
    ///
    /// CHECK: This account will be read via zero-copy deserialization in the instruction
    /// handler, which will determine which type of VAA account is being used. If this account
    /// is the legacy PostedVaaV1 account, its PDA address will be verified by this zero-copy
    /// reader.
    #[account(owner = core_bridge::id())]
    vaa: AccountInfo<'info>,

    /// Claim account (mut), which acts as replay protection after consuming data from the VAA
    /// account.
    ///
    /// Seeds: [emitter_address, emitter_chain, sequence],
    /// seeds::program = token_bridge_program.
    ///
    /// CHECK: This account is created via [claim_vaa](core_bridge_program::sdk::claim_vaa).
    /// This account can only be created once for this VAA.
    #[account(mut)]
    claim: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [try_decree(&vaa, |decree| decree.foreign_chain())?.to_be_bytes().as_ref()],
        bump,
    )]
    registered_emitter: Account<'info, core_bridge::legacy::LegacyAnchorized<RegisteredEmitter>>,

    /// This account should be created using only the emitter chain ID as its seed. Instead, it uses
    /// both emitter chain and address to derive this PDA address. Having both of these as seeds
    /// potentially allows for multiple emitters to be registered for a given chain ID (when there
    /// should only be one).
    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [
            try_decree(&vaa, |decree| decree.foreign_chain())?.to_be_bytes().as_ref(),
            try_decree(&vaa, |decree| decree.foreign_emitter())?.as_ref(),
        ],
        bump,
    )]
    legacy_registered_emitter:
        Account<'info, core_bridge::legacy::LegacyAnchorized<RegisteredEmitter>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa_acc_info = &ctx.accounts.vaa;
        let vaa_key = vaa_acc_info.key();
        let vaa = core_bridge::VaaAccount::load(vaa_acc_info)?;
        let gov_payload = crate::processor::require_valid_governance_vaa(&vaa_key, &vaa)?;

        gov_payload
            .register_chain()
            .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))?;

        // Done.
        Ok(())
    }
}

#[access_control(RegisterChain::constraints(&ctx))]
pub fn register_chain(ctx: Context<RegisterChain>) -> Result<()> {
    let vaa = core_bridge::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge::claim_vaa(
        CpiContext::new(
            ctx.accounts.system_program.to_account_info(),
            core_bridge::ClaimVaa {
                claim: ctx.accounts.claim.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
            },
        ),
        &crate::ID,
        &vaa,
        None,
    )?;

    // Deserialize and set data in registered emitter accounts.
    {
        let gov_payload = TokenBridgeGovPayload::try_from(vaa.try_payload().unwrap())
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

    // Done.
    Ok(())
}

fn try_decree<F, T>(vaa_acc_info: &AccountInfo, func: F) -> Result<T>
where
    F: FnOnce(&wormhole_raw_vaas::token_bridge::RegisterChain) -> T,
{
    let vaa = core_bridge::VaaAccount::load(vaa_acc_info)?;
    let gov_payload = TokenBridgeGovPayload::try_from(vaa.try_payload()?)
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;
    gov_payload
        .decree()
        .register_chain()
        .map(func)
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}
