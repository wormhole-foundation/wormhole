#![allow(unexpected_cfgs)]

declare_id!("GbFfTqMqKDgAMRH8VmDmoLTdvDd1853TnkkEwpydv3J6");

mod vaa;
mod append_threshold_key_message;
mod threshold_key;
mod utils;
mod hex_literal;

use anchor_lang::{prelude::*, Result};

use wormhole_anchor_sdk::wormhole::{program::Wormhole, constants::CHAIN_ID_SOLANA, PostedVaaData};

use vaa::{VAA, VAABody};
use append_threshold_key_message::AppendThresholdKeyMessage;
use threshold_key::{
  AppendThresholdKeyError,
  ThresholdKeyAccount,
  init_threshold_key_account
};
use utils::{SeedPrefix};
use hex_literal::hex;

const GOVERNANCE_ADDRESS: [u8; 32] =
  hex!("0000000000000000000000000000000000000000000000000000000000000004");

#[error_code]
pub enum VAAError {
  #[msg("Invalid VAA version")]
  InvalidVersion,
  #[msg("Invalid VAA index")]
  InvalidIndex,
  TSSKeyExpired,
  BodyTooSmall,
}

#[account]
#[derive(InitSpace)]
pub struct LatestKeyAccount {
  pub account: Pubkey,
}

impl SeedPrefix for LatestKeyAccount {
  const SEED_PREFIX: &'static [u8] = b"latestkey";
}

// TODO: Refactor this to have one accounts/instruction per file
#[derive(Accounts)]
pub struct InitThresholdKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    owner = wormhole_program.key() @ AppendThresholdKeyError::InvalidVAA,
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA
      @ AppendThresholdKeyError::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS
      @ AppendThresholdKeyError::InvalidGovernanceAddress,
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

  /// CHECK: See `init_threshold_key_account` for checks on this account.
  #[account(mut)]
  pub new_threshold_key: UncheckedAccount<'info>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct AppendThresholdKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    owner = wormhole_program.key() @ AppendThresholdKeyError::InvalidVAA,
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA
      @ AppendThresholdKeyError::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS
      @ AppendThresholdKeyError::InvalidGovernanceAddress,
  )]
  pub vaa: Account<'info, PostedVaaData>,

  #[account(mut)]
  pub latest_key: Account<'info, LatestKeyAccount>,

  /// CHECK: See `init_threshold_key_account` for checks on this account.
  #[account(mut)]
  pub new_threshold_key: UncheckedAccount<'info>,

  #[account(
    mut,
    constraint = old_threshold_key.key() == latest_key.account
      @ AppendThresholdKeyError::InvalidOldThresholdKey,
  )]
  pub old_threshold_key: Account<'info, ThresholdKeyAccount>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct VerifyVaa<'info> {
  #[account(
    constraint = threshold_key.is_unexpired() @ VAAError::TSSKeyExpired,
  )]
  pub threshold_key: Account<'info, ThresholdKeyAccount>,
}

#[program]
pub mod verification_v2 {
  use super::*;

  pub fn init_threshold_key(ctx: Context<InitThresholdKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendThresholdKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    init_threshold_key_account(
      ctx.accounts.new_threshold_key.to_account_info(),
      message.tss_index,
      message.tss_key,
      &ctx.accounts.system_program,
      ctx.accounts.payer.to_account_info()
    )?;

    ctx.accounts.latest_key.account = ctx.accounts.new_threshold_key.key();

    Ok(())
  }

  pub fn append_threshold_key(ctx: Context<AppendThresholdKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendThresholdKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    let old_threshold_key = &mut ctx.accounts.old_threshold_key;

    // Check that the index is increasing from the previous index
    if message.tss_index <= old_threshold_key.index {
      return Err(AppendThresholdKeyError::InvalidNewKeyIndex.into());
    }

    init_threshold_key_account(
      ctx.accounts.new_threshold_key.to_account_info(),
      message.tss_index,
      message.tss_key,
      &ctx.accounts.system_program,
      ctx.accounts.payer.to_account_info()
    )?;

    old_threshold_key.update_expiration_timestamp(message.expiration_delay_seconds as u64);

    ctx.accounts.latest_key.account = ctx.accounts.new_threshold_key.key();

    Ok(())
  }

  pub fn verify_vaa(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<()> {
    verify_vaa_impl(ctx, raw_vaa)?;
    Ok(())
  }

  pub fn verify_vaa_and_decode(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<VAABody> {
    let body_buf = verify_vaa_impl(ctx, raw_vaa)?;
    let body = VAABody::deserialize(&mut body_buf.as_slice())?;
    Ok(body)
  }
}

fn verify_vaa_impl(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<Vec<u8>> {
  let vaa = VAA::deserialize(&mut raw_vaa.as_slice())?;

  vaa.check_valid()?;

  let threshold_key = &ctx.accounts.threshold_key;
  if threshold_key.index != vaa.header.tss_index {
    return Err(VAAError::InvalidIndex.into());
  }

  let msg_hash = vaa.message_hash()?;

  threshold_key.tss_key.check_signature(&msg_hash, &vaa.header.signature)?;

  Ok(vaa.body)
}