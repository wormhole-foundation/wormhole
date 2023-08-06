use crate::{
    constants::UPGRADE_SEED_PREFIX, error::TokenBridgeError, legacy::instruction::EmptyArgs,
    state::Claim,
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, sdk::cpi::CoreBridge,
    zero_copy::PostedVaaV1,
};
use solana_program::{bpf_loader_upgradeable, program::invoke_signed};

#[derive(Accounts)]
pub struct UpgradeContract<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump,
        seeds::program = core_bridge_program,
    )]
    posted_vaa: AccountInfo<'info>,

    /// Account representing that a VAA has been consumed.
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
    claim: Account<'info, LegacyAnchorized<0, Claim>>,

    /// CHECK: We need this upgrade authority to invoke the BPF Loader Upgradeable program to
    /// upgrade this program's executable. We verify this PDA address here out of convenience to get
    /// the PDA bump seed to invoke the upgrade.
    #[account(
        seeds = [UPGRADE_SEED_PREFIX],
        bump,
    )]
    upgrade_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
    #[account(mut)]
    spill: UncheckedAccount<'info>,

    /// CHECK: Deployed implementation. The pubkey of this account is checked in access control
    /// against the one encoded in the governance VAA.
    buffer: AccountInfo<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
    program_data: UncheckedAccount<'info>,

    /// CHECK: Unnecessary account.
    _this_program: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: BPF Loader Upgradeable program.
    #[account(address = solana_program::bpf_loader_upgradeable::id())]
    bpf_loader_upgradeable_program: AccountInfo<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
}

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for UpgradeContract<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyUpgradeContract";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = upgrade_contract;
}

impl<'info> UpgradeContract<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = &ctx.accounts.posted_vaa;
        let vaa_key = vaa.key();
        let acc_data: &[u8] = &vaa.try_borrow_data()?;
        let gov_payload = super::require_valid_posted_governance_vaa(&vaa_key, acc_data)?;

        let decree = gov_payload
            .contract_upgrade()
            .ok_or(error!(TokenBridgeError::InvalidGovernanceAction))?;

        // Make sure that the contract upgrade is intended for this network.
        require_eq!(
            decree.chain(),
            SOLANA_CHAIN,
            TokenBridgeError::GovernanceForAnotherChain
        );

        // Read the implementation pubkey and check against the buffer in our account context.
        require_keys_eq!(
            Pubkey::from(decree.implementation()),
            ctx.accounts.buffer.key(),
            TokenBridgeError::ImplementationMismatch
        );

        // Done.
        Ok(())
    }
}

/// Processor for contract upgrade governance decrees. This instruction handler invokes the BPF
/// Loader Upgradeable program to upgrade this program's executable to the provided buffer.
#[access_control(UpgradeContract::constraints(&ctx))]
fn upgrade_contract(ctx: Context<UpgradeContract>, _args: EmptyArgs) -> Result<()> {
    // Mark the claim as complete. The account only exists to ensure that the VAA is not processed,
    // so this value does not matter. But the legacy program set this data to true.
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
