use crate::{
    constants::SOLANA_CHAIN,
    error::CoreBridgeError,
    legacy::{instruction::EmptyArgs, utils::LegacyAnchorized},
    state::Config,
    zero_copy::{LoadZeroCopy, VaaAccount},
};
use anchor_lang::prelude::*;
use ruint::aliases::U256;
use wormhole_raw_vaas::core::CoreBridgeGovPayload;

#[derive(Accounts)]
pub struct SetMessageFee<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// For governance VAAs, we need to make sure that the current guardian set was used to attest
    /// for this governance decree.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    vaa: AccountInfo<'info>,

    /// CHECK: Account representing that a VAA has been consumed. Seeds are checked when
    /// [claim_vaa](crate::utils::vaa::claim_vaa) is called.
    #[account(mut)]
    claim: AccountInfo<'info>,

    system_program: Program<'info, System>,
}

impl<'info> crate::utils::cpi::CreateAccount<'info> for SetMessageFee<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for SetMessageFee<'info>
{
    const LOG_IX_NAME: &'static str = "LegacySetMessageFee";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = set_message_fee;
}

impl<'info> SetMessageFee<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = VaaAccount::load(&ctx.accounts.vaa)?;
        let gov_payload = super::require_valid_governance_vaa(&ctx.accounts.config, &vaa)?;

        let decree = gov_payload
            .set_message_fee()
            .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))?;

        // Make sure that setting the message fee is intended for this network.
        require_eq!(
            decree.chain(),
            SOLANA_CHAIN,
            CoreBridgeError::GovernanceForAnotherChain
        );

        // Make sure that the encoded fee does not overflow since the encoded amount is u256 (and
        // lamports are u64).
        let fee = U256::from_be_bytes(decree.fee());
        require!(fee <= U256::from(u64::MAX), CoreBridgeError::U64Overflow);

        // Done.
        Ok(())
    }
}

/// Processor for setting Wormhole message fee governance decrees. This instruction handler changes
/// the message fee in the [Config] account.
#[access_control(SetMessageFee::constraints(&ctx))]
fn set_message_fee(ctx: Context<SetMessageFee>, _args: EmptyArgs) -> Result<()> {
    let vaa = VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    crate::utils::vaa::claim_vaa(ctx.accounts, &ctx.accounts.claim, &crate::ID, &vaa, None)?;

    let gov_payload = CoreBridgeGovPayload::try_from(vaa.try_payload().unwrap())
        .unwrap()
        .decree();

    // Uint encodes limbs in little endian, so we will take the first u64 value.
    let fee = U256::from_be_bytes(gov_payload.set_message_fee().unwrap().fee());
    ctx.accounts.config.fee_lamports = fee.as_limbs()[0];

    // Done.
    Ok(())
}
