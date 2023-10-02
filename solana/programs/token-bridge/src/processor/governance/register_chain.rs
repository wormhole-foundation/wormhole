use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::{
    legacy::utils::LegacyAnchorized,
    sdk::{self as core_bridge_sdk, LoadZeroCopy},
};
use wormhole_raw_vaas::token_bridge::TokenBridgeGovPayload;

#[derive(Accounts)]
pub struct RegisterChain<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(mut)]
    vaa: AccountInfo<'info>,

    /// CHECK: Account representing that a VAA has been consumed. Seeds are checked when
    /// [claim_vaa](core_bridge_sdk::cpi::claim_vaa) is called.
    #[account(mut)]
    claim: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = RegisteredEmitter::INIT_SPACE,
        seeds = [try_decree(&vaa, |decree| decree.foreign_chain())?.to_be_bytes().as_ref()],
        bump,
    )]
    registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

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
    legacy_registered_emitter: Account<'info, LegacyAnchorized<0, RegisteredEmitter>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info> for RegisterChain<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> RegisterChain<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa_acc_info = &ctx.accounts.vaa;
        let vaa_key = vaa_acc_info.key();
        let vaa = core_bridge_sdk::VaaAccount::load(vaa_acc_info)?;
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
    let vaa = core_bridge_sdk::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge_sdk::cpi::claim_vaa(ctx.accounts, &ctx.accounts.claim, &crate::ID, &vaa)?;

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
    where F: FnOnce(&wormhole_raw_vaas::token_bridge::RegisterChain) -> T {
    let vaa = core_bridge_sdk::VaaAccount::load(vaa_acc_info)?;
    let gov_payload = TokenBridgeGovPayload::try_from(vaa.try_payload()?)
        .map_err(|_| error!(TokenBridgeError::InvalidGovernanceVaa))?;
    gov_payload
        .decree()
        .register_chain()
        .map(func)
        .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))
}
