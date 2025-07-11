#![allow(unexpected_cfgs)]

declare_id!("GbFfTqMqKDgAMRH8VmDmoLTdvDd1853TnkkEwpydv3J6");

mod vaa;
mod append_schnorr_key_message;
mod schnorr_key;
mod utils;
mod hex_literal;

use anchor_lang::{prelude::*, Result};

use wormhole_anchor_sdk::wormhole::{program::Wormhole, constants::CHAIN_ID_SOLANA, PostedVaaData};

use vaa::{VAA, VAAHeader};
use append_schnorr_key_message::AppendSchnorrKeyMessage;
use schnorr_key::{
  AppendSchnorrKeyError,
  SchnorrKey,
  SchnorrKeyAccount,
  init_schnorr_key_account
};
use hex_literal::hex;

const GOVERNANCE_ADDRESS: [u8; 32] =
  hex!("0000000000000000000000000000000000000000000000000000000000000004");

#[error_code]
pub enum VAAError {
  #[msg("Invalid VAA version")]
  InvalidVersion,
  #[msg("Invalid VAA index")]
  InvalidIndex,
  SchnorrKeyExpired,
  BodyTooSmall,
}

#[account]
#[derive(InitSpace)]
pub struct LatestKeyAccount {
  pub account: Pubkey,
}

impl LatestKeyAccount {
  const SEED_PREFIX: &'static [u8] = b"latestkey";
}

// TODO: Refactor this to have one accounts/instruction per file
#[derive(Accounts)]
pub struct InitSchnorrKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    owner = wormhole_program.key() @ AppendSchnorrKeyError::InvalidVAA,
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA
      @ AppendSchnorrKeyError::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS
      @ AppendSchnorrKeyError::InvalidGovernanceAddress,
  )]
  pub vaa: Account<'info, PostedVaaData>,

  #[account(
    init,
    payer = payer,
    space = 8 + LatestKeyAccount::INIT_SPACE,
    seeds = [LatestKeyAccount::SEED_PREFIX],
    bump
  )]
  pub latest_key: Account<'info, LatestKeyAccount>,

  /// CHECK: See `init_schnorr_key_account` for checks on this account.
  #[account(mut)]
  pub new_schnorr_key: UncheckedAccount<'info>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct AppendSchnorrKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    owner = wormhole_program.key() @ AppendSchnorrKeyError::InvalidVAA,
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA
      @ AppendSchnorrKeyError::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS
      @ AppendSchnorrKeyError::InvalidGovernanceAddress,
  )]
  pub vaa: Account<'info, PostedVaaData>,

  #[account(mut)]
  pub latest_key: Account<'info, LatestKeyAccount>,

  /// CHECK: See `init_schnorr_key_account` for checks on this account.
  #[account(mut)]
  pub new_schnorr_key: UncheckedAccount<'info>,

  #[account(
    mut,
    constraint = old_schnorr_key.key() == latest_key.account
      @ AppendSchnorrKeyError::InvalidOldSchnorrKey,
  )]
  pub old_schnorr_key: Account<'info, SchnorrKeyAccount>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct VerifyVaa<'info> {
  #[account(
    constraint = schnorr_key.is_unexpired() @ VAAError::SchnorrKeyExpired,
  )]
  pub schnorr_key: Account<'info, SchnorrKeyAccount>,
}

#[program]
pub mod verification_v2 {
  use super::*;

  pub fn init_schnorr_key(ctx: Context<InitSchnorrKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendSchnorrKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    init_schnorr_key_account(
      ctx.accounts.new_schnorr_key.to_account_info(),
      message.schnorr_key_index,
      message.schnorr_key,
      &ctx.accounts.system_program,
      ctx.accounts.payer.to_account_info()
    )?;

    ctx.accounts.latest_key.account = ctx.accounts.new_schnorr_key.key();

    Ok(())
  }

  pub fn append_schnorr_key(ctx: Context<AppendSchnorrKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendSchnorrKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    let old_schnorr_key = &mut ctx.accounts.old_schnorr_key;

    // Check that the index is increasing from the previous index
    if message.schnorr_key_index <= old_schnorr_key.index {
      return Err(AppendSchnorrKeyError::InvalidNewKeyIndex.into());
    }

    init_schnorr_key_account(
      ctx.accounts.new_schnorr_key.to_account_info(),
      message.schnorr_key_index,
      message.schnorr_key,
      &ctx.accounts.system_program,
      ctx.accounts.payer.to_account_info()
    )?;

    old_schnorr_key.update_expiration_timestamp(message.expiration_delay_seconds as u64);

    ctx.accounts.latest_key.account = ctx.accounts.new_schnorr_key.key();

    Ok(())
  }

  pub fn verify_vaa(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<()> {
    verify_vaa_impl(ctx, raw_vaa)?;
    Ok(())
  }

  pub fn verify_vaa_and_decode(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<Vec<u8>> {
    let body = verify_vaa_impl(ctx, raw_vaa)?;
    Ok(body)
  }

  pub fn verify_vaa_header_with_digest(
    ctx: Context<VerifyVaa>,
    raw_vaa_header: [u8; VAAHeader::SIZE],
    digest: [u8; SchnorrKey::DIGEST_SIZE]
  ) -> Result<()> {
    let vaa_header = VAAHeader::deserialize(&mut raw_vaa_header.as_slice())?;

    vaa_header.check_valid()?;

    let schnorr_key_account = &ctx.accounts.schnorr_key;
    if schnorr_key_account.index != vaa_header.schnorr_key_index {
      return Err(VAAError::InvalidIndex.into());
    }

    schnorr_key_account.schnorr_key.check_signature(&digest, &vaa_header.signature)?;

    Ok(())
  }
}

fn verify_vaa_impl(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<Vec<u8>> {
  let vaa = VAA::deserialize(&mut raw_vaa.as_slice())?;

  vaa.check_valid()?;

  let schnorr_key_account = &ctx.accounts.schnorr_key;
  if schnorr_key_account.index != vaa.header.schnorr_key_index {
    return Err(VAAError::InvalidIndex.into());
  }

  let digest = vaa.digest()?;

  schnorr_key_account.schnorr_key.check_signature(&digest.to_bytes(), &vaa.header.signature)?;

  Ok(vaa.body)
}