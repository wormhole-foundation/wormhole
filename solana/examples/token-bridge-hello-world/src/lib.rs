#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use anchor_spl::token;
use wormhole_token_bridge_solana::sdk as token_bridge;

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
        seeds = [token_bridge::PROGRAM_REDEEMER_SEED_PREFIX],
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
    token_bridge_program: Program<'info, token_bridge::TokenBridge>,
    token_program: Program<'info, token::Token>,
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
        seeds = [token_bridge::PROGRAM_SENDER_SEED_PREFIX],
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
    token_bridge_program: Program<'info, token_bridge::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

#[program]
pub mod token_bridge_hello_world {
    use super::*;

    pub fn redeem_hello_world(_ctx: Context<RedeemHelloWorld>) -> Result<()> {
        // token_bridge_sdk::cpi::complete_transfer(
        //     ctx.accounts,
        //     Some(&[&[
        //         token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX,
        //         &[ctx.bumps["redeemer_authority"]],
        //     ]]),
        // )
        Ok(())
    }

    pub fn transfer_hello_world(_ctx: Context<TransferHelloWorld>, _amount: u64) -> Result<()> {
        let _nonce = 420;
        let _redeemer = [
            0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xde, 0xad, 0xbe, 0xef,
            0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad,
            0xbe, 0xef,
        ];
        let _redeemer_chain = 2;
        let _payload = b"Hello, world!".to_vec();

        // token_bridge_sdk::cpi::transfer_tokens(
        //     ctx.accounts,
        //     token_bridge_sdk::cpi::TransferTokensDirective::ProgramTransferWithPayload {
        //         program_id: crate::ID,
        //         nonce,
        //         amount,
        //         redeemer,
        //         redeemer_chain,
        //         payload,
        //     },
        //     Some(&[&[
        //         token_bridge_sdk::PROGRAM_SENDER_SEED_PREFIX,
        //         &[ctx.bumps["sender_authority"]],
        //     ]]),
        // )

        Ok(())
    }
}
