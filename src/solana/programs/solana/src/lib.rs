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
use wormhole_anchor_sdk::wormhole::{SEED_PREFIX_POSTED_VAA, PostedVaaData};

use vaa::VAA;
use append_threshold_key_message::AppendThresholdKeyMessage;

const GOVERNANCE_ADDRESS: [u8; 32] = [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4];

#[derive(Accounts)]
#[instruction(new_index: u32, vaa_hash: [u8; 32])]
pub struct AppendThresholdKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    seeds = [
      SEED_PREFIX_POSTED_VAA,
      &vaa_hash
    ],
    bump,
    seeds::program = wormhole_program.key
  )]
  pub vaa: Account<'info, PostedVaaData>,

  #[account(
    init,
    payer = payer,
    space = 8 + ThresholdKey::INIT_SPACE,
    seeds = [b"threshold_key", new_index.to_be_bytes().as_ref()],
    bump,
  )]
  pub new_threshold_key: Account<'info, ThresholdKey>,

  #[account(
    seeds = [b"threshold_key", &((new_index - 1).to_be_bytes())],
    bump,
  )]
  pub old_threshold_key: Option<Account<'info, ThresholdKey>>,

  pub wormhole_program: Program<'info, Wormhole>,
  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(vaa: Vec<u8>)]
pub struct VerifyVaa<'info> {
  #[account(
    seeds = [b"threshold_key", VAA::get_seed_bytes(&vaa)],
    bump,
  )]
  pub threshold_key: Account<'info, ThresholdKey>,
}

#[account]
#[derive(InitSpace)]
pub struct ThresholdKey {
  pub bump: u8,
  pub index: u32, // TODO: Is this needed?
  pub key: [u8; 20],
  pub expiration: u64,
}

#[error_code]
pub enum VAAError {
  #[msg("Invalid VAA version")]
  InvalidVersion,
  #[msg("Invalid VAA index")]
  InvalidIndex,
  #[msg("Guardian set expired")]
  GuardianSetExpired,
  #[msg("Invalid signature")]
  InvalidSignature,
  #[msg("Invalid governance chain ID")]
  InvalidGovernanceChainId,
  #[msg("Invalid governance address")]
  InvalidGovernanceAddress,
}

#[error_code]
pub enum AppendThresholdKeyError {
  #[msg("Invalid governance chain ID")]
  InvalidGovernanceChainId,
  #[msg("Invalid governance address")]
  InvalidGovernanceAddress,
  #[msg("Index mismatch")]
  IndexMismatch,
  #[msg("Invalid old threshold key")]
  InvalidOldThresholdKey,
}

#[program]
pub mod verification_v2 {
  use super::*;

  // TODO: Get the new index from the VAA?
  pub fn append_threshold_key(ctx: Context<AppendThresholdKey>, new_index: u32, _vaa_hash: [u8; 32]) -> Result<()> {
    // Validate the VAA metadata
    let vaa = &ctx.accounts.vaa;
    if vaa.meta.emitter_chain != CHAIN_ID_SOLANA {
      return Err(AppendThresholdKeyError::InvalidGovernanceChainId.into());
    }

    if vaa.meta.emitter_address != GOVERNANCE_ADDRESS {
      return Err(AppendThresholdKeyError::InvalidGovernanceAddress.into());
    }

    // Decode the message
    let message = AppendThresholdKeyMessage::deserialize(&vaa.payload)?;

    // Check the index matches the VAA index
    if new_index != message.guardian_set_index {
      return Err(AppendThresholdKeyError::IndexMismatch.into());
    }

    // Validate the old threshold key
    if new_index == 0 {
      if ctx.accounts.old_threshold_key.is_some() {
        return Err(AppendThresholdKeyError::InvalidOldThresholdKey.into());
      }
    } else {
      if let Some(old_threshold_key) = &ctx.accounts.old_threshold_key {
        if old_threshold_key.index != new_index - 1 {
          return Err(AppendThresholdKeyError::InvalidOldThresholdKey.into());
        }
      } else {
        return Err(AppendThresholdKeyError::InvalidOldThresholdKey.into());
      }
    }

    // Set the new threshold key
    ctx.accounts.new_threshold_key.index = new_index;
    ctx.accounts.new_threshold_key.key = message.guardian_set_key;
    ctx.accounts.new_threshold_key.expiration = 0;

    // Set the old threshold key
    if let Some(ref mut old_threshold_key) = &mut ctx.accounts.old_threshold_key {
      let current_timestamp = Clock::get().unwrap().unix_timestamp as u64;
      old_threshold_key.expiration = current_timestamp + message.expiration_delay_seconds as u64;
    }

    Ok(())
  }

  pub fn verify_vaa(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<VAA> {
    // Decode the VAA
    let (vaa, double_hash) = VAA::deserialize(&raw_vaa)?;

    // Check if the VAA version is valid
    if vaa.version != 2 {
      return Err(VAAError::InvalidVersion.into());
    }

    // Check if the threshold key index matches the VAA index
    let threshold_key = &mut ctx.accounts.threshold_key;
    if threshold_key.index != vaa.guardian_set_index {
      return Err(VAAError::InvalidIndex.into());
    }

    // Check if the threshold key has expired
    if threshold_key.expiration != 0 && threshold_key.expiration < Clock::get().unwrap().unix_timestamp as u64 {
      return Err(VAAError::GuardianSetExpired.into());
    }

    // Verify the VAA signature
    let message = Message::parse(&double_hash);
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
