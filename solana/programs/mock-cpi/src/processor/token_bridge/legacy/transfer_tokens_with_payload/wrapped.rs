use crate::{constants::MESSAGE_SEED_PREFIX, state::SignerSequence};
use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::sdk as token_bridge;

use super::{MockLegacyTransferTokensWithPayloadArgs, CUSTOM_SENDER_SEED_PREFIX};

#[derive(Accounts)]
pub struct MockLegacyTransferTokensWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = 8 + SignerSequence::INIT_SPACE,
        seeds = [SignerSequence::SEED_PREFIX, payer.key().as_ref()],
        bump,
    )]
    payer_sequence: Account<'info, SignerSequence>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [token_bridge::PROGRAM_SENDER_SEED_PREFIX],
        bump,
    )]
    token_bridge_program_sender_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [CUSTOM_SENDER_SEED_PREFIX],
        bump,
    )]
    token_bridge_custom_sender_authority: Option<AccountInfo<'info>>,

    #[account(
        mut,
        token::mint = token_bridge_wrapped_mint,
        token::authority = payer
    )]
    src_token: Account<'info, token::TokenAccount>,

    #[account(mut)]
    token_bridge_wrapped_mint: Account<'info, token::Mint>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_wrapped_asset: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_transfer_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        mut,
        seeds = [
            MESSAGE_SEED_PREFIX,
            payer.key().as_ref(),
            payer_sequence.to_le_bytes().as_ref()
        ],
        bump,
    )]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_fee_collector: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

pub fn mock_legacy_transfer_tokens_with_payload_wrapped(
    ctx: Context<MockLegacyTransferTokensWithPayloadWrapped>,
    args: MockLegacyTransferTokensWithPayloadArgs,
) -> Result<()> {
    let MockLegacyTransferTokensWithPayloadArgs {
        nonce,
        amount,
        redeemer,
        redeemer_chain,
        payload,
    } = args;

    // We are determining which sender authority to test. A program can either use his own program
    // ID as the sender address or a custom sender address (like his PDA).
    let (sender_authority, cpi_program_id, sender_seed_prefix, sender_bump) = match (
        &ctx.accounts.token_bridge_program_sender_authority,
        &ctx.accounts.token_bridge_custom_sender_authority,
    ) {
        (Some(authority), _) => (
            authority,
            Some(crate::ID),
            token_bridge::PROGRAM_SENDER_SEED_PREFIX,
            ctx.bumps["token_bridge_program_sender_authority"],
        ),
        (None, Some(authority)) => (
            authority,
            None,
            CUSTOM_SENDER_SEED_PREFIX,
            ctx.bumps["token_bridge_custom_sender_authority"],
        ),
        (None, None) => return err!(ErrorCode::AccountNotEnoughKeys),
    };

    let sequence_number = ctx.accounts.payer_sequence.take_and_uptick();

    token_bridge::transfer_tokens_with_payload_wrapped(
        CpiContext::new_with_signer(
            ctx.accounts.token_bridge_program.to_account_info(),
            token_bridge::TransferTokensWithPayloadWrapped {
                payer: ctx.accounts.payer.to_account_info(),
                src_token: ctx.accounts.src_token.to_account_info(),
                wrapped_mint: ctx.accounts.token_bridge_wrapped_mint.to_account_info(),
                wrapped_asset: ctx.accounts.token_bridge_wrapped_asset.to_account_info(),
                transfer_authority: ctx
                    .accounts
                    .token_bridge_transfer_authority
                    .to_account_info(),
                core_bridge_config: ctx.accounts.core_bridge_config.to_account_info(),
                core_message: ctx.accounts.core_message.to_account_info(),
                core_emitter: ctx.accounts.core_emitter.to_account_info(),
                core_emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                core_fee_collector: ctx.accounts.core_fee_collector.clone(),
                sender_authority: sender_authority.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
                token_program: ctx.accounts.token_program.to_account_info(),
                core_bridge_program: ctx.accounts.core_bridge_program.to_account_info(),
            },
            &[
                &[
                    MESSAGE_SEED_PREFIX,
                    ctx.accounts.payer.key().as_ref(),
                    sequence_number.as_ref(),
                    &[ctx.bumps["core_message"]],
                ],
                &[sender_seed_prefix, &[sender_bump]],
            ],
        ),
        token_bridge::TransferTokensWithPayloadArgs {
            nonce,
            amount,
            redeemer,
            redeemer_chain,
            payload,
            cpi_program_id,
        },
    )
}
