#![allow(unexpected_cfgs)]

declare_id!("GbFfTqMqKDgAMRH8VmDmoLTdvDd1853TnkkEwpydv3J6");

mod vaa;
mod append_threshold_key_message;
mod threshold_key;

use anchor_lang::prelude::*;
use anchor_lang::solana_program::clock::Clock;

use wormhole_anchor_sdk::wormhole::program::Wormhole;
use wormhole_anchor_sdk::wormhole::constants::CHAIN_ID_SOLANA;
use wormhole_anchor_sdk::wormhole::{PostedVaaData};

use vaa::VAA;
use append_threshold_key_message::AppendThresholdKeyMessage;
use threshold_key::ThresholdKey;

const GOVERNANCE_ADDRESS: [u8; 32] = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4];

#[error_code]
pub enum VAAError {
  #[msg("Invalid VAA version")]
  InvalidVersion,
  #[msg("Invalid VAA index")]
  InvalidIndex,
  #[msg("TSS key expired")]
  TSSKeyExpired,
  #[msg("Invalid signature")]
  InvalidSignature,
}

#[error_code]
pub enum AppendThresholdKeyError {
  #[msg("Invalid VAA")]
  InvalidVAA,
  #[msg("Invalid governance chain ID")]
  InvalidGovernanceChainId,
  #[msg("Invalid governance address")]
  InvalidGovernanceAddress,
  #[msg("Index mismatch")]
  IndexMismatch,
  #[msg("Invalid old threshold key")]
  InvalidOldThresholdKey,
  #[msg("Invalid TSS key")]
  InvalidTSSKey,
}

#[account]
#[derive(InitSpace)]
pub struct ThresholdKeyAccount {
  pub index: u32,
  pub key: ThresholdKey,
  pub expiration_timestamp: u64,
}

impl ThresholdKeyAccount {
  pub fn is_unexpired(&self) -> bool {
    self.expiration_timestamp == 0 || self.expiration_timestamp > Clock::get().unwrap().unix_timestamp as u64
  }

  pub fn update_expiration_timestamp(&mut self, time_lapse: u64) {
    let current_timestamp = Clock::get().unwrap().unix_timestamp as u64;
    self.expiration_timestamp = current_timestamp + time_lapse;
  }
}

// TODO: Refactor this to have one accounts/instruction per file
#[derive(Accounts)]
pub struct AppendThresholdKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    owner = wormhole_program.key() @ AppendThresholdKeyError::InvalidVAA,
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA @ AppendThresholdKeyError::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS @ AppendThresholdKeyError::InvalidGovernanceAddress,
  )]
  pub vaa: Account<'info, PostedVaaData>,

  #[account(
    init,
    payer = payer,
    space = 8 + ThresholdKeyAccount::INIT_SPACE,
  )]
  pub new_threshold_key: Account<'info, ThresholdKeyAccount>,

  #[account(
    mut,
    constraint = old_threshold_key.expiration_timestamp == 0 @ AppendThresholdKeyError::InvalidOldThresholdKey,
  )]
  pub old_threshold_key: Option<Account<'info, ThresholdKeyAccount>>,

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

  pub fn append_threshold_key(ctx: Context<AppendThresholdKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendThresholdKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    // Check that if there is no old threshold key, the index is 0
    // Otherwise, check that the index is increasing from the previous index
    // FIXME: There's nothing preventing us from creating many chains of valid threshold keys unless we use PDAs based on index
    let expected_index = ctx.accounts.old_threshold_key.as_ref().map_or(0, |key| key.index + 1);
    if message.tss_index != expected_index {
      return Err(AppendThresholdKeyError::IndexMismatch.into());
    }

    // Set the new threshold key
    ctx.accounts.new_threshold_key.index = message.tss_index;
    ctx.accounts.new_threshold_key.key = message.tss_key;
    ctx.accounts.new_threshold_key.expiration_timestamp = 0;

    // Set the old threshold key expiration timestamp
    if let Some(ref mut old_threshold_key) = &mut ctx.accounts.old_threshold_key {
      old_threshold_key.update_expiration_timestamp(message.expiration_delay_seconds as u64);
    } else if expected_index > 0 {
      return Err(AppendThresholdKeyError::InvalidOldThresholdKey.into());
    }

    Ok(())
  }

  pub fn verify_vaa(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<VAA> {
    // Decode the VAA
    let vaa = VAA::deserialize(&mut raw_vaa.as_slice())?;

    // Check that the threshold key index matches the VAA index
    let threshold_key = &mut ctx.accounts.threshold_key;
    if threshold_key.index != vaa.header.tss_index {
      return Err(VAAError::InvalidIndex.into());
    }

    // Check that the signature is valid
    threshold_key.key.check_signature(&vaa.message_hash()?, &vaa.header.signature)?;

    // Return the VAA
    Ok(vaa)
  }
}
