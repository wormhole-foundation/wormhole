mod governance;
pub use governance::*;

// mod transfer_tokens;
// pub use transfer_tokens::*;

// mod transfer_tokens_with_payload;
// pub use transfer_tokens_with_payload::*;

use crate::{
    constants::{
        CUSTODY_AUTHORITY_SEED_PREFIX, EMITTER_SEED_PREFIX, MINT_AUTHORITY_SEED_PREFIX,
        TRANSFER_AUTHORITY_SEED_PREFIX,
    },
    error::TokenBridgeError,
    utils::TruncateAmount,
};
use anchor_lang::{prelude::*, system_program};
use anchor_spl::token::{self, Burn, Mint, MintTo, Token, TokenAccount};
use core_bridge_program::{state::Config as CoreBridgeConfig, types::Commitment, CoreBridge};
use wormhole_io::Writeable;

pub struct PostTokenBridgeMessage<'ctx, 'info> {
    pub core_bridge_config: &'ctx Account<'info, CoreBridgeConfig>,
    pub core_message: &'ctx AccountInfo<'info>,
    pub core_emitter: &'ctx AccountInfo<'info>,
    pub core_emitter_sequence: &'ctx AccountInfo<'info>,
    pub payer: &'ctx Signer<'info>,
    pub core_fee_collector: &'ctx Option<UncheckedAccount<'info>>,
    pub system_program: &'ctx Program<'info, System>,
    pub core_bridge_program: &'ctx Program<'info, CoreBridge>,
}

pub fn post_token_bridge_message<W: Writeable>(
    accounts: PostTokenBridgeMessage<'_, '_>,
    emitter_bump: u8,
    nonce: u32,
    message: W,
) -> Result<()> {
    // Pay fee to the core bridge program if there is one.
    let fee_collector = match (
        accounts.core_bridge_config.fee_lamports,
        accounts.core_fee_collector,
    ) {
        (0, _) => None,
        (lamports, Some(core_fee_collector)) => {
            system_program::transfer(
                CpiContext::new(
                    accounts.system_program.to_account_info(),
                    system_program::Transfer {
                        from: accounts.payer.to_account_info(),
                        to: core_fee_collector.to_account_info(),
                    },
                ),
                lamports,
            )?;

            Some(core_fee_collector.to_account_info())
        }
        _ => return err!(TokenBridgeError::CoreFeeCollectorRequired),
    };

    let mut payload = Vec::with_capacity(message.written_size());
    message.write(&mut payload)?;

    core_bridge_program::legacy::cpi::post_message(
        CpiContext::new_with_signer(
            accounts.core_bridge_program.to_account_info(),
            core_bridge_program::legacy::cpi::PostMessage {
                config: accounts.core_bridge_config.to_account_info(),
                message: accounts.core_message.to_account_info(),
                emitter: accounts.core_emitter.to_account_info(),
                emitter_sequence: accounts.core_emitter_sequence.to_account_info(),
                payer: accounts.payer.to_account_info(),
                fee_collector,
                system_program: accounts.system_program.to_account_info(),
            },
            &[&[EMITTER_SEED_PREFIX, &[emitter_bump]]],
        ),
        core_bridge_program::legacy::cpi::PostMessageArgs {
            nonce,
            payload: message.to_vec(),
            commitment: Commitment::Finalized,
        },
    )
}

pub fn mint_wrapped_tokens<'info>(
    token_program: &Program<'info, Token>,
    wrapped_mint: &Account<'info, Mint>,
    dst_token: &Account<'info, TokenAccount>,
    mint_authority: &AccountInfo<'info>,
    mint_authority_bump: u8,
    mint_amount: u64,
) -> Result<()> {
    token::mint_to(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            MintTo {
                mint: wrapped_mint.to_account_info(),
                to: dst_token.to_account_info(),
                authority: mint_authority.to_account_info(),
            },
            &[&[MINT_AUTHORITY_SEED_PREFIX, &[mint_authority_bump]]],
        ),
        mint_amount,
    )
}

pub fn burn_wrapped_tokens<'info>(
    token_program: &Program<'info, Token>,
    wrapped_mint: &Account<'info, Mint>,
    src_token: &Account<'info, TokenAccount>,
    transfer_authority: &AccountInfo<'info>,
    transfer_authority_bump: u8,
    burn_amount: u64,
) -> Result<()> {
    token::burn(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            Burn {
                mint: wrapped_mint.to_account_info(),
                from: src_token.to_account_info(),
                authority: transfer_authority.to_account_info(),
            },
            &[&[TRANSFER_AUTHORITY_SEED_PREFIX, &[transfer_authority_bump]]],
        ),
        burn_amount,
    )
}

pub fn withdraw_native_tokens<'info>(
    token_program: &Program<'info, Token>,
    custody_token: &Account<'info, TokenAccount>,
    dst_token: &Account<'info, TokenAccount>,
    custody_authority: &AccountInfo<'info>,
    custody_authority_bump: u8,
    transfer_amount: u64,
) -> Result<()> {
    token::transfer(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            token::Transfer {
                from: custody_token.to_account_info(),
                to: dst_token.to_account_info(),
                authority: custody_authority.to_account_info(),
            },
            &[&[CUSTODY_AUTHORITY_SEED_PREFIX, &[custody_authority_bump]]],
        ),
        transfer_amount,
    )
}

pub fn deposit_native_tokens<'info>(
    token_program: &Program<'info, Token>,
    mint: &Account<'info, Mint>,
    src_token: &Account<'info, TokenAccount>,
    custody_token: &Account<'info, TokenAccount>,
    transfer_authority: &AccountInfo<'info>,
    transfer_authority_bump: u8,
    raw_amount: u64,
) -> Result<u64> {
    let transfer_amount = mint.truncate_amount(raw_amount);

    token::transfer(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            token::Transfer {
                from: src_token.to_account_info(),
                to: custody_token.to_account_info(),
                authority: transfer_authority.to_account_info(),
            },
            &[&[TRANSFER_AUTHORITY_SEED_PREFIX, &[transfer_authority_bump]]],
        ),
        transfer_amount,
    )?;

    Ok(transfer_amount)
}
