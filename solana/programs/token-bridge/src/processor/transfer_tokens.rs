use crate::{
    constants::{EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::TransferTokensArgs,
    processor::{burn_wrapped_tokens, post_token_bridge_message},
    state::WrappedAsset,
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, Token, TokenAccount};
use core_bridge_program::{
    self,
    state::{Config as CoreBridgeConfig, EmitterSequence},
    CoreBridge,
};
use ruint::aliases::U256;
use solana_program::program_option::COption;

#[derive(Accounts)]
pub struct TransferTokens<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction. This account is a
    /// placeholder just in case we can use a Token Bridge program configuration.
    _config: UncheckedAccount<'info>,

    #[account(mut)]
    src_token: Box<Account<'info, TokenAccount>>,

    /// CHECK: This account can either be Token Bridge's custody token account or Token Bridge's
    /// wrapped mint account. Either way, this account must be mutable.
    #[account(mut)]
    custody_token_or_wrapped_mint: AccountInfo<'info>,

    /// CHECK: This account can either be a native SPL mint or Token Bridge's wrapped asset account.
    /// Either way, this account is read-only.
    native_mint_or_wrapped_asset: AccountInfo<'info>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// We need to deserialize this account to determine the Wormhole message fee.
    #[account(
        mut,
        seeds = [CoreBridgeConfig::SEED_PREFIX],
        bump,
        seeds::program = core_bridge_program
    )]
    core_bridge_config: Account<'info, CoreBridgeConfig>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(
        mut,
        seeds = [b"msg", core_emitter_sequence.to_be_bytes().as_ref()],
        bump,
    )]
    core_message: AccountInfo<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// We let the Core Bridge program validate this PDA. But we deserialize this account for the
    /// core message PDA.
    #[account(mut)]
    core_emitter_sequence: Account<'info, EmitterSequence>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
    token_program: Program<'info, Token>,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TransferRelayable {
    pub nonce: u32,
    pub amount: u64,
    pub relayer_fee: u64,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TransferWithPayload {
    pub nonce: u32,
    pub amount: u64,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub payload: Vec<u8>,
    pub cpi_program_id: Option<Pubkey>,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum TransferTokensDirective {
    TransferRelayable(TransferRelayable),
    TransferWithPayload(TransferWithPayload),
}

pub fn transfer_tokens(
    ctx: Context<TransferTokens>,
    directive: TransferTokensDirective,
) -> Result<()> {
    match directive {
        TransferTokensDirective::TransferRelayable(args) => relayable(ctx, args),
        TransferTokensDirective::TransferWithPayload(args) => unimplemented!(),
    }
}

fn relayable(ctx: Context<TransferTokens>, args: TransferRelayable) -> Result<()> {
    let TransferRelayable {
        nonce,
        amount,
        relayer_fee,
        recipient,
        recipient_chain,
    } = args;

    require_gte!(amount, relayer_fee, TokenBridgeError::InvalidRelayerFee);

    let token_transfer = if is_wrapped(&ctx.accounts.custody_token_or_wrapped_mint) {
        // Chicken and egg problem here. First we will deserialize the wrapped asset account so we
        // can derive the wrapped mint PDA address. Then we will validate the wrapped asset PDA
        // address with this wrapped mint pubkey.
        let mut wrapped_asset = {
            let info = &ctx.accounts.native_mint_or_wrapped_asset;
            require_keys_eq!(info.owner, crate::ID, ErrorCode::ConstraintOwner);

            WrappedAsset::try_deserialize(&mut &info.try_borrow_data()?)?
        };

        // Burn wrapped assets from the sender's account.
        burn_wrapped_tokens(
            &ctx.accounts.token_program,
            &ctx.accounts.wrapped_mint,
            &ctx.accounts.src_token,
            &ctx.accounts.transfer_authority,
            ctx.bumps["transfer_authority"],
            amount,
        )?;

        // Prepare Wormhole message. Amounts do not need to be normalized because we are working with
        // wrapped assets.
        crate::messages::Transfer {
            norm_amount: U256::from(amount),
            token_address: wrapped_asset.token_address,
            token_chain: wrapped_asset.token_chain,
            recipient,
            recipient_chain,
            norm_relayer_fee: U256::from(relayer_fee),
        }
    } else {
        // Deposit native assets from the sender's account into the custody account.
        let amount = deposit_native_tokens(
            &ctx.accounts.token_program,
            &ctx.accounts.mint,
            &ctx.accounts.src_token,
            &ctx.accounts.custody_token,
            &ctx.accounts.transfer_authority,
            ctx.bumps["transfer_authority"],
            amount,
        )?;
    };

    // Finally publish Wormhole message using the Core Bridge.
    post_token_bridge_message(
        ctx.accounts,
        ctx.bumps["core_emitter"],
        nonce,
        token_transfer,
    )
}

fn is_wrapped(mint: &Account<'_, Mint>) -> bool {
    if let COption::Some(mint_authority) = mint.mint_authority {
        let (token_bridge_mint_authority, _) = Pubkey::find_program_address(
            &[crate::constants::MINT_AUTHORITY_SEED_PREFIX],
            &crate::ID,
        );

        if mint_authority == token_bridge_mint_authority {
            true
        } else {
            false
        }
    } else {
        false
    }
}
