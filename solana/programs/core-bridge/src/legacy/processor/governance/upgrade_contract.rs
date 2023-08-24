use crate::{
    constants::{SOLANA_CHAIN, UPGRADE_SEED_PREFIX},
    error::CoreBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, Config},
    zero_copy::PostedVaaV1,
};
use anchor_lang::prelude::*;
use solana_program::{bpf_loader_upgradeable, program::invoke_signed};
use wormhole_solana_common::{BpfLoaderUpgradeable, SeedPrefix};

#[derive(Accounts)]
pub struct UpgradeContract<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, Config>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump
    )]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_address().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_chain().to_be_bytes().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.sequence().to_be_bytes().as_ref(),
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    /// CHECK: We need this upgrade authority to invoke the BPF Loader Upgradeable program to
    /// upgrade this program's executable.
    #[account(
        seeds = [UPGRADE_SEED_PREFIX],
        bump,
    )]
    upgrade_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
    #[account(mut)]
    spill: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
    ///
    /// NOTE: This account's pubkey is what is encoded in the governance VAA. We check this in the
    /// instruction handler.
    buffer: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
    program_data: UncheckedAccount<'info>,

    /// CHECK: Unnecessary account.
    _this_program: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    bpf_loader_upgradeable_program: Program<'info, BpfLoaderUpgradeable>,
    system_program: Program<'info, System>,
}

impl<'info> UpgradeContract<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let acc_data = ctx.accounts.posted_vaa.try_borrow_data()?;
        let gov_payload =
            super::require_valid_posted_governance_vaa(&acc_data, &ctx.accounts.config)?;

        let decree = gov_payload
            .contract_upgrade()
            .ok_or(error!(CoreBridgeError::InvalidGovernanceAction))?;

        require_eq!(
            decree.chain(),
            SOLANA_CHAIN,
            CoreBridgeError::GovernanceForAnotherChain
        );

        // Read the implementation pubkey and check against the buffer in our account context.
        require_keys_eq!(
            Pubkey::from(decree.implementation()),
            ctx.accounts.buffer.key(),
            CoreBridgeError::ImplementationMismatch
        );

        // Done.
        Ok(())
    }
}

#[access_control(UpgradeContract::constraints(&ctx))]
pub fn upgrade_contract(ctx: Context<UpgradeContract>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // Finally upgrade.
    invoke_signed(
        &bpf_loader_upgradeable::upgrade(
            &crate::ID,
            &ctx.accounts.buffer.key(),
            &ctx.accounts.upgrade_authority.key(),
            &ctx.accounts.spill.key(),
        ),
        &ctx.accounts.to_account_infos(),
        &[&[UPGRADE_SEED_PREFIX, &[ctx.bumps["upgrade_authority"]]]],
    )
    .map_err(Into::into)
}
