use crate::{constants::MESSAGE_SEED_PREFIX, state::SignerSequence};
use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::sdk::{self as token_bridge_sdk, core_bridge_sdk};

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
        seeds = [token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX],
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
    #[account(mut)]
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
    core_fee_collector: Option<UncheckedAccount<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for MockLegacyTransferTokensWithPayloadWrapped<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info>
    for MockLegacyTransferTokensWithPayloadWrapped<'info>
{
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        self.core_fee_collector
            .as_ref()
            .map(|acc| acc.to_account_info())
    }
}
impl<'info> token_bridge_sdk::cpi::TransferTokens<'info>
    for MockLegacyTransferTokensWithPayloadWrapped<'info>
{
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn core_message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.token_bridge_wrapped_mint.to_account_info()
    }

    fn src_token_account(&self) -> AccountInfo<'info> {
        self.src_token.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn token_bridge_transfer_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_transfer_authority.to_account_info()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        Some(self.token_bridge_wrapped_asset.to_account_info())
    }

    fn sender_authority(&self) -> Option<AccountInfo<'info>> {
        match (
            &self.token_bridge_program_sender_authority,
            &self.token_bridge_custom_sender_authority,
        ) {
            (Some(sender_authority), _) => Some(sender_authority.to_account_info()),
            (None, Some(sender_authority)) => Some(sender_authority.to_account_info()),
            (None, None) => None,
        }
    }
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
    let (directive, sender_seed_prefix, sender_bump) = match (
        &ctx.accounts.token_bridge_program_sender_authority,
        &ctx.accounts.token_bridge_custom_sender_authority,
    ) {
        (Some(_), _) => (
            token_bridge_sdk::cpi::TransferTokensDirective::ProgramTransferWithPayload {
                program_id: crate::ID,
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
            },
            token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX,
            ctx.bumps["token_bridge_program_sender_authority"],
        ),
        (None, Some(_)) => (
            token_bridge_sdk::cpi::TransferTokensDirective::SignerTransferWithPayload {
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
            },
            CUSTOM_SENDER_SEED_PREFIX,
            ctx.bumps["token_bridge_custom_sender_authority"],
        ),
        (None, None) => return err!(ErrorCode::AccountNotEnoughKeys),
    };

    let sequence_number = ctx.accounts.payer_sequence.take_and_uptick();

    token_bridge_sdk::cpi::transfer_tokens_specified(
        ctx.accounts,
        directive,
        true, // is_wrapped_asset
        Some(&[
            &[
                MESSAGE_SEED_PREFIX,
                ctx.accounts.payer.key().as_ref(),
                sequence_number.as_ref(),
                &[ctx.bumps["core_message"]],
            ],
            &[sender_seed_prefix, &[sender_bump]],
        ]),
    )
}
