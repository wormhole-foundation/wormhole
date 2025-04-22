#![allow(unexpected_cfgs)]

declare_id!("CTyCJvaLgY18BTTY3M1ga1SmY5fuT3Jyu3j7s332Z3oz");

mod vaa;
mod append_threshold_key_message;

use anchor_lang::prelude::*;
use anchor_lang::solana_program::clock::Clock;
use anchor_lang::solana_program::keccak::hash;

use libsecp256k1::{Message, RecoveryId, Signature, recover};

use wormhole_anchor_sdk::wormhole::program::Wormhole;
use wormhole_anchor_sdk::wormhole::constants::CHAIN_ID_SOLANA;
use wormhole_anchor_sdk::wormhole::{PostedVaaData};

use vaa::VAA;
use append_threshold_key_message::AppendThresholdKeyMessage;

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
}

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
    space = 8 + ThresholdKey::INIT_SPACE,
  )]
  pub new_threshold_key: Account<'info, ThresholdKey>,

  #[account(
    mut,
    constraint = old_threshold_key.expiration_timestamp == 0 @ AppendThresholdKeyError::InvalidOldThresholdKey,
  )]
  pub old_threshold_key: Option<Account<'info, ThresholdKey>>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct VerifyVaa<'info> {
  #[account(
    constraint = threshold_key.is_unexpired() @ VAAError::TSSKeyExpired,
  )]
  pub threshold_key: Account<'info, ThresholdKey>,
}

#[account]
#[derive(InitSpace)]
pub struct ThresholdKey {
  pub index: u32,
  pub key: [u8; 20],
  pub expiration_timestamp: u64,
}

impl ThresholdKey {
  pub fn is_unexpired(&self) -> bool {
    self.expiration_timestamp == 0 || self.expiration_timestamp > Clock::get().unwrap().unix_timestamp as u64
  }

  pub fn update_expiration_timestamp(&mut self, new_expiration_timestamp: u64) {
    let current_timestamp = Clock::get().unwrap().unix_timestamp as u64;
    self.expiration_timestamp = current_timestamp + new_expiration_timestamp;
  }
}

#[program]
pub mod verification_v2 {
  use super::*;

  pub fn append_threshold_key(ctx: Context<AppendThresholdKey>) -> Result<()> {
    // Decode the VAA payload
    let message = AppendThresholdKeyMessage::deserialize(&ctx.accounts.vaa.payload)?;

    // Validate the message index
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
    }

    Ok(())
  }

  pub fn verify_vaa(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<VAA> {
    // Decode the VAA
    let (vaa, vaa_hash) = VAA::deserialize(&raw_vaa)?;

    // Check if the VAA version is valid
    if vaa.version != 2 {
      return Err(VAAError::InvalidVersion.into());
    }

    // Check if the threshold key index matches the VAA index
    let threshold_key = &mut ctx.accounts.threshold_key;
    if threshold_key.index != vaa.tss_index {
      return Err(VAAError::InvalidIndex.into());
    }

    // Verify the VAA signature
    let message = Message::parse(&vaa_hash);
    let signature = Signature::parse_standard(&vaa.signature).map_err(|_| VAAError::InvalidSignature)?;
    let recovery_id = RecoveryId::parse(vaa.recovery_id).map_err(|_| VAAError::InvalidSignature)?;
    let recovered_key = recover(&message, &signature, &recovery_id).map_err(|_| VAAError::InvalidSignature)?;
    let recovered_eth_key = &hash(&recovered_key.serialize()[1..]).to_bytes()[12..];

    if recovered_eth_key != threshold_key.key {
      return Err(VAAError::InvalidSignature.into());
    }

    // Return the VAA
    Ok(vaa)
  }
}
