#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use anchor_spl::token;
use wormhole_token_bridge_solana::sdk::{self as token_bridge_sdk, core_bridge_sdk};

declare_id!("TokenBridgeHe11oWor1d1111111111111111111111");

#[derive(Accounts)]
pub struct TransferHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account.
    #[account(owner = token::ID)]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge transfer with payload. This PDA
    /// validates the sender address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX],
        bump,
    )]
    sender_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_transfer_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_core_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> core_bridge_sdk::cpi::CreateAccount<'info> for TransferHelloWorld<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter(&self) -> Option<AccountInfo<'info>> {
        Some(self.token_bridge_core_emitter.to_account_info())
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}

impl<'info> token_bridge_sdk::cpi::TransferTokens<'info> for TransferHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn core_message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn src_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn token_bridge_transfer_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_transfer_authority.to_account_info()
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }

    fn token_bridge_sender_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.sender_authority.to_account_info())
    }
}

#[program]
pub mod token_bridge_hello_world {
    use super::*;

    pub fn transfer_hello_world(ctx: Context<TransferHelloWorld>, amount: u64) -> Result<()> {
        let nonce = 420;
        let redeemer = [
            0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xde, 0xad, 0xbe, 0xef,
            0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad,
            0xbe, 0xef,
        ];
        let redeemer_chain = 2;
        let payload = b"Hello, world!".to_vec();

        token_bridge_sdk::cpi::transfer_tokens(
            ctx.accounts,
            token_bridge_sdk::cpi::TransferTokensDirective::ProgramTransferWithPayload {
                program_id: crate::ID,
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
            },
            Some(&[&[
                token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX,
                &[ctx.bumps["sender_authority"]],
            ]]),
        )
    }
}
