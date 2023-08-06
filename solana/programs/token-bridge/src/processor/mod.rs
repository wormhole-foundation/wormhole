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
    utils::TruncateAmount,
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::{sdk as core_bridge_sdk, types::Commitment};
use wormhole_io::Writeable;

pub fn post_token_bridge_message<
    'info,
    I: core_bridge_sdk::cpi::PublishMessage<'info>,
    W: Writeable,
>(
    accounts: &I,
    emitter_bump: u8,
    nonce: u32,
    message: W,
) -> Result<()> {
    core_bridge_sdk::cpi::publish_message(
        accounts,
        core_bridge_sdk::cpi::PublishMessageDirective::Message {
            nonce,
            payload: message.to_vec(),
            commitment: Commitment::Finalized,
        },
        &[EMITTER_SEED_PREFIX, &[emitter_bump]],
        None,
    )
}

pub fn mint_wrapped_tokens<'info>(
    token_program: &Program<'info, token::Token>,
    wrapped_mint: &AccountInfo<'info>,
    dst_token: &AccountInfo<'info>,
    mint_authority: &AccountInfo<'info>,
    mint_authority_bump: u8,
    mint_amount: u64,
) -> Result<()> {
    token::mint_to(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            token::MintTo {
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
    token_program: &Program<'info, token::Token>,
    wrapped_mint: &AccountInfo<'info>,
    src_token: &AccountInfo<'info>,
    transfer_authority: &AccountInfo<'info>,
    transfer_authority_bump: u8,
    burn_amount: u64,
) -> Result<()> {
    token::burn(
        CpiContext::new_with_signer(
            token_program.to_account_info(),
            token::Burn {
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
    token_program: &Program<'info, token::Token>,
    custody_token: &AccountInfo<'info>,
    dst_token: &AccountInfo<'info>,
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
    token_program: &Program<'info, token::Token>,
    mint: &AccountInfo<'info>,
    src_token: &AccountInfo<'info>,
    custody_token: &Account<'info, token::TokenAccount>,
    transfer_authority: &AccountInfo<'info>,
    transfer_authority_bump: u8,
    raw_amount: u64,
) -> Result<u64> {
    let transfer_amount = Mint::parse(&mint.data.borrow())
        .unwrap()
        .truncate_amount(raw_amount);

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
