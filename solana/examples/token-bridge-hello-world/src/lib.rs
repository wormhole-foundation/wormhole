#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use anchor_spl::token;
use wormhole_token_bridge_solana::sdk::{self as token_bridge_sdk, core_bridge_sdk};

declare_id!("TokenBridgeHe11oWor1d1111111111111111111111");

#[derive(Accounts)]
pub struct RedeemHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        associated_token::mint = mint,
        associated_token::authority = payer,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint of our token account. This account is mutable in case the transferred asset is
    /// Token Bridge wrapped, which requires the Token Bridge program to mint to a token account.
    #[account(
        mut,
        owner = token::ID
    )]
    mint: AccountInfo<'info>,

    /// CHECK: This account acts as the signer for our Token Bridge complete transfer with payload.
    /// This PDA validates the redeemer address as this program's ID.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    redeemer_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This account warehouses the
    /// VAA of the token transfer from another chain.
    encoded_vaa: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_claim: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_registered_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    #[account(mut)]
    token_bridge_custody_token_account: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// native tokens.
    token_bridge_custody_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_wrapped_asset: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program. This should not be None for
    /// wrapped tokens.
    token_bridge_mint_authority: Option<AccountInfo<'info>>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> token_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for RedeemHelloWorld<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> token_bridge_sdk::cpi::CompleteTransfer<'info> for RedeemHelloWorld<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn vaa(&self) -> AccountInfo<'info> {
        self.encoded_vaa.to_account_info()
    }

    fn token_bridge_claim(&self) -> AccountInfo<'info> {
        self.token_bridge_claim.to_account_info()
    }

    fn token_bridge_registered_emitter(&self) -> AccountInfo<'info> {
        self.token_bridge_registered_emitter.to_account_info()
    }

    fn dst_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn redeemer_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.redeemer_authority.to_account_info())
    }

    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_authority.clone()
    }

    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_custody_token_account.clone()
    }

    fn token_bridge_mint_authority(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_mint_authority.clone()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset.clone()
    }
}

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

    /// CHECK: Mint of our token account. This account is mutable in case the transferred asset is
    /// Token Bridge wrapped, which requires the Token Bridge program to burn from a token
    /// account.
    #[account(
        mut,
        owner = token::ID
    )]
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
    #[account(mut)]
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

impl<'info> token_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for TransferHelloWorld<'info>
{
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

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.token_bridge_core_emitter.to_account_info()
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

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn core_message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn src_token_account(&self) -> AccountInfo<'info> {
        self.payer_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.mint.to_account_info()
    }

    fn sender_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.sender_authority.to_account_info())
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
}

#[program]
pub mod token_bridge_hello_world {
    use super::*;

    pub fn redeem_hello_world(ctx: Context<RedeemHelloWorld>) -> Result<()> {
        token_bridge_sdk::cpi::complete_transfer(
            ctx.accounts,
            Some(&[&[
                token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX,
                &[ctx.bumps["redeemer_authority"]],
            ]]),
        )
    }

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
