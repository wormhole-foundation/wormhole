use crate::constants::EMITTER_SEED_PREFIX;
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::sdk as core_bridge_sdk;
use wormhole_io::Writeable;

pub trait MintTo<'info> {
    fn token_program(&self) -> AccountInfo<'info>;

    fn mint(&self) -> AccountInfo<'info>;

    fn mint_authority(&self) -> AccountInfo<'info>;
}

pub fn mint_to<'info, A>(
    accounts: &A,
    to: &AccountInfo<'info>,
    mint_amount: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: MintTo<'info>,
{
    token::mint_to(
        CpiContext::new_with_signer(
            accounts.token_program(),
            token::MintTo {
                mint: accounts.mint(),
                to: to.to_account_info(),
                authority: accounts.mint_authority(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        mint_amount,
    )
}

pub trait Burn<'info> {
    fn token_program(&self) -> AccountInfo<'info>;

    fn mint(&self) -> AccountInfo<'info>;

    fn from(&self) -> Option<AccountInfo<'info>> {
        None
    }

    fn authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    fn require_from(&self) -> Result<AccountInfo<'info>> {
        self.from().ok_or(error!(ErrorCode::AccountNotEnoughKeys))
    }

    fn require_authority(&self) -> Result<AccountInfo<'info>> {
        self.authority()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))
    }
}

pub fn burn<'info, A>(
    accounts: &A,
    burn_amount: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: Burn<'info>,
{
    burn_from(
        accounts,
        accounts
            .from()
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?,
        accounts
            .authority()
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?,
        burn_amount,
        signer_seeds,
    )
}

fn burn_from<'info, A>(
    accounts: &A,
    from: &AccountInfo<'info>,
    authority: &AccountInfo<'info>,
    burn_amount: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: Burn<'info>,
{
    token::burn(
        CpiContext::new_with_signer(
            accounts.token_program(),
            token::Burn {
                mint: accounts.mint(),
                from: from.to_account_info(),
                authority: authority.to_account_info(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        burn_amount,
    )
}

pub trait Transfer<'info> {
    fn token_program(&self) -> AccountInfo<'info>;

    fn from(&self) -> Option<AccountInfo<'info>> {
        None
    }

    fn authority(&self) -> Option<AccountInfo<'info>> {
        None
    }
}

pub fn transfer<'info, A>(
    accounts: &A,
    to: &AccountInfo<'info>,
    transfer_amount: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: Transfer<'info>,
{
    transfer_from(
        accounts,
        accounts
            .from()
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?,
        accounts
            .authority()
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?,
        to,
        transfer_amount,
        signer_seeds,
    )
}

pub fn transfer_from<'info, A>(
    accounts: &A,
    from: &AccountInfo<'info>,
    authority: &AccountInfo<'info>,
    to: &AccountInfo<'info>,
    transfer_amount: u64,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: Transfer<'info>,
{
    token::transfer(
        CpiContext::new_with_signer(
            accounts.token_program(),
            token::Transfer {
                from: from.to_account_info(),
                to: to.to_account_info(),
                authority: authority.to_account_info(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        transfer_amount,
    )
}

pub fn post_token_bridge_message<'info, A, W>(
    accounts: &A,
    core_message: &AccountInfo<'info>,
    nonce: u32,
    message: W,
) -> Result<()>
where
    A: core_bridge_sdk::cpi::PublishMessage<'info>,
    W: Writeable,
{
    // Validate core emitter pubkey.
    let (expected_core_emitter, emitter_bump) =
        Pubkey::find_program_address(&[EMITTER_SEED_PREFIX], &crate::ID);
    require_keys_eq!(
        accounts.core_emitter_authority().key(),
        expected_core_emitter,
        ErrorCode::ConstraintSeeds,
    );

    core_bridge_sdk::cpi::publish_message(
        accounts,
        core_message,
        core_bridge_sdk::cpi::PublishMessageDirective::Message {
            nonce,
            payload: message.to_vec(),
            commitment: core_bridge_sdk::types::Commitment::Finalized,
        },
        Some(&[&[EMITTER_SEED_PREFIX, &[emitter_bump]]]),
    )
}
