use crate::{
    constants::EMITTER_SEED_PREFIX, error::TokenBridgeError, legacy::instruction::AttestTokenArgs,
    state::WrappedAsset, utils,
};
use anchor_lang::prelude::*;
use anchor_spl::{metadata, token};
use core_bridge_program::sdk as core_bridge;
use solana_program::program_memory::sol_memcpy;

#[derive(Accounts)]
pub struct AttestToken<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// Previously needed config account.
    ///
    /// CHECK: This account is unchecked.
    _config: UncheckedAccount<'info>,

    /// CHECK: Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: Account<'info, token::Mint>,

    /// Non-existent wrapped asset account.
    ///
    /// CHECK: This account should have zero data. Otherwise this mint was created by the Token
    /// Bridge program.
    #[account(
        seeds = [
            WrappedAsset::SEED_PREFIX,
            mint.key().as_ref()
        ],
        bump,
        constraint = non_existent_wrapped_asset.data_is_empty() @ TokenBridgeError::WrappedAsset
    )]
    non_existent_wrapped_asset: AccountInfo<'info>,

    /// We derive this PDA because we do not involve the Token Metadata program with this
    /// instruction handler. It is the Token Bridge's job to verify that the metadata attested for
    /// is the correct one.
    #[account(
        seeds = [
            b"metadata",
            metadata::Metadata::id().as_ref(),
            mint.key().as_ref()
        ],
        bump,
        seeds::program = metadata::Metadata::id()
    )]
    token_metadata: Account<'info, metadata::MetadataAccount>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    /// This PDA address is checked in `publish_token_bridge_message`.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<AccountInfo<'info>>,

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _clock: UncheckedAccount<'info>,

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
}

impl<'info> core_bridge::legacy::ProcessLegacyInstruction<'info, AttestTokenArgs>
    for AttestToken<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyAttestToken";

    const ANCHOR_IX_FN: fn(Context<Self>, AttestTokenArgs) -> Result<()> = attest_token;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        utils::fix_account_order(
            account_infos,
            11,       // start_index
            12,       // system_program_index
            None,     // token_program_index
            Some(13), // core_bridge_program_index
        )
    }
}

fn attest_token(ctx: Context<AttestToken>, args: AttestTokenArgs) -> Result<()> {
    let AttestTokenArgs { nonce } = args;

    let metadata = &ctx.accounts.token_metadata.data;

    // Finally post Wormhole message via Core Bridge.
    utils::cpi::publish_token_bridge_message(
        CpiContext::new_with_signer(
            ctx.accounts.core_bridge_program.to_account_info(),
            core_bridge::PublishMessage {
                payer: ctx.accounts.payer.to_account_info(),
                message: ctx.accounts.core_message.to_account_info(),
                emitter_authority: ctx.accounts.core_emitter.to_account_info(),
                config: ctx.accounts.core_bridge_config.to_account_info(),
                emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                fee_collector: ctx.accounts.core_fee_collector.clone(),
                system_program: ctx.accounts.system_program.to_account_info(),
            },
            &[&[EMITTER_SEED_PREFIX, &[ctx.bumps["core_emitter"]]]],
        ),
        nonce,
        crate::messages::Attestation {
            token_address: ctx.accounts.mint.key().to_bytes(),
            token_chain: core_bridge::SOLANA_CHAIN,
            decimals: ctx.accounts.mint.decimals,
            symbol: string_to_fixed32(&metadata.symbol),
            name: string_to_fixed32(&metadata.name),
        },
    )
}

pub(crate) fn string_to_fixed32(s: &String) -> [u8; 32] {
    let mut bytes = [0; 32];
    if s.len() > 32 {
        sol_memcpy(&mut bytes, s.as_bytes(), 32);
    } else {
        sol_memcpy(&mut bytes, s.as_bytes(), s.len());
    }
    bytes
}
