use std::io;

use crate::{
    constants::UPGRADE_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    message::{require_valid_governance_posted_vaa, TokenBridgeGovernance},
    state::Claim,
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    message::PostedGovernanceVaaV1, state::VaaV1LegacyAccount, WormDecode, WormEncode,
};
use solana_program::{bpf_loader_upgradeable, program::invoke_signed};
use wormhole_common::{BpfLoaderUpgradeable, SeedPrefix};

#[derive(Debug, Clone)]
struct UpgradeContractDecree {
    implementation: Pubkey,
}

impl WormDecode for UpgradeContractDecree {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let implementation = Pubkey::decode_reader(reader)?;
        Ok(Self { implementation })
    }
}

impl WormEncode for UpgradeContractDecree {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.implementation.encode(writer)
    }
}

#[derive(Accounts)]
pub struct UpgradeContract<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        seeds = [
            PostedGovernanceVaaV1::<UpgradeContractDecree>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedGovernanceVaaV1<UpgradeContractDecree>>,

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

    /// CHECK: We need this upgrade authority to invoke the BPF Loader Upgradeable program to
    /// upgrade this program's executable.
    #[account(
        seeds = [UPGRADE_SEED_PREFIX],
        bump,
    )]
    upgrade_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the BPF Loader Upgradeable program.
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
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let msg = require_valid_governance_posted_vaa(&ctx.accounts.posted_vaa)?;

        // We expect a specific governance header.
        require!(
            msg.header == TokenBridgeGovernance::ContractUpgrade.try_into()?,
            TokenBridgeError::InvalidGovernanceAction
        );

        // Read the implementation pubkey and check against the buffer in our account context.
        require_keys_eq!(msg.decree.implementation, ctx.accounts.buffer.key());

        // Done.
        Ok(())
    }
}

#[access_control(UpgradeContract::accounts(&ctx))]
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
