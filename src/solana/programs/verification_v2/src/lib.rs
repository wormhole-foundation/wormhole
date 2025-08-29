#![allow(unexpected_cfgs)]

declare_id!("GbFfTqMqKDgAMRH8VmDmoLTdvDd1853TnkkEwpydv3J6");

mod vaa;
mod append_schnorr_key_message;
mod schnorr_key;
mod hex_literal;

use anchor_lang::prelude::*;
#[cfg(feature = "idl-build")]
use anchor_lang::IdlBuild;

use wormhole_anchor_sdk::wormhole::{constants::CHAIN_ID_SOLANA, PostedVaa, SignatureSetData};

use vaa::{VAA, VAAHeader, VAASchnorrSignature};
use append_schnorr_key_message::AppendSchnorrKeyMessage;
use schnorr_key::SchnorrKeyAccount;
use hex_literal::hex;

const GOVERNANCE_ADDRESS: [u8; 32] =
  hex!("0000000000000000000000000000000000000000000000000000000000000004");

pub const DIGEST_SIZE: usize = 32;

#[error_code]
pub enum VerificationV2Error {
  InvalidGovernanceChainId,
  InvalidGovernanceAddress,
  InvalidSignatureSet,
  InvalidGuardianSet,
  InvalidOldSchnorrKey,
  SchnorrKeyExpired,
  InvalidAccounts,
  NewKeyIndexNotDirectSuccessor,
  IndexMismatch,
}

#[account]
#[derive(InitSpace)]
pub struct LatestKeyAccount {
  pub account: Pubkey,
}

impl LatestKeyAccount {
  const SEED_PREFIX: &'static [u8] = b"latestkey";
}

#[derive(Accounts)]
pub struct AppendSchnorrKey<'info> {
  #[account(mut)]
  pub payer: Signer<'info>,

  #[account(
    constraint = vaa.meta.emitter_chain == CHAIN_ID_SOLANA
      @ VerificationV2Error::InvalidGovernanceChainId,
    constraint = vaa.meta.emitter_address == GOVERNANCE_ADDRESS
      @ VerificationV2Error::InvalidGovernanceAddress,
    constraint = vaa.meta.signature_set == signature_set.key()
      @ VerificationV2Error::InvalidSignatureSet,
  )]
  pub vaa: Account<'info, PostedVaa::<AppendSchnorrKeyMessage>>,

  /// CHECK: need to deserialize manually because Anchor 0.31.1 does not support empty discriminators.
  pub signature_set: UncheckedAccount<'info>,

  #[account(
    init_if_needed,
    payer = payer,
    space = 8 + LatestKeyAccount::INIT_SPACE,
    seeds = [LatestKeyAccount::SEED_PREFIX],
    bump
  )]
  pub latest_key: Account<'info, LatestKeyAccount>,

  #[account(
    init,
    payer = payer,
    space = 8 + SchnorrKeyAccount::INIT_SPACE,
    seeds = [SchnorrKeyAccount::SEED_PREFIX, &vaa.data().schnorr_key_index.to_le_bytes()],
    bump
  )]
  pub new_schnorr_key: Account<'info, SchnorrKeyAccount>,

  #[account(
    mut,
    constraint = old_schnorr_key.key() == latest_key.account
      @ VerificationV2Error::InvalidOldSchnorrKey,
  )]
  pub old_schnorr_key: Option<Account<'info, SchnorrKeyAccount>>,

  pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct VerifyVaa<'info> {
  #[account(
    constraint = schnorr_key.is_unexpired() @ VerificationV2Error::SchnorrKeyExpired,
  )]
  pub schnorr_key: Account<'info, SchnorrKeyAccount>,
}

#[program]
pub mod verification_v2 {
  use super::*;

  pub fn append_schnorr_key(ctx: Context<AppendSchnorrKey>) -> Result<()> {
    require_eq!( //either both must be true (initialization) or both must be false (append)
      (ctx.accounts.latest_key.account == Pubkey::default()),
      ctx.accounts.old_schnorr_key.is_none(),
      VerificationV2Error::InvalidAccounts,
    );

    // ctx.accounts.signature_set.try_borrow_data()?
    let buf = ctx.accounts.signature_set.try_borrow_data()?;
    let signature_set_data: Vec<u8> = buf.to_vec();
    let signature_set: SignatureSetData = AccountDeserialize::try_deserialize_unchecked(&mut signature_set_data.as_slice())?;
    require_gte!(
      signature_set.guardian_set_index,
      ctx.accounts.vaa.data().expected_mss_index,
      VerificationV2Error::InvalidGuardianSet,
    );

    // Check that the index is increasing from the previous index
    if let Some(old_schnorr_key) = ctx.accounts.old_schnorr_key.as_deref_mut() {
      require_eq!(
        old_schnorr_key.index + 1,
        ctx.accounts.vaa.data().schnorr_key_index,
        VerificationV2Error::NewKeyIndexNotDirectSuccessor,
      );

      old_schnorr_key.update_expiration_timestamp(
        ctx.accounts.vaa.data().expiration_delay_seconds as u64
      );
    }

    ctx.accounts.new_schnorr_key.set_inner(SchnorrKeyAccount {
      index: ctx.accounts.vaa.data().schnorr_key_index,
      schnorr_key: ctx.accounts.vaa.data().schnorr_key.clone(),
      expiration_timestamp: 0,
    });

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

  #[inline(always)]
  pub fn verify_vaa_header_with_digest(
    ctx: Context<VerifyVaa>,
    raw_vaa_header: [u8; VAAHeader::SIZE],
    digest: [u8; DIGEST_SIZE]
  ) -> Result<()> {
    let vaa_header = VAAHeader::deserialize(&mut raw_vaa_header.as_slice())?;
    verify(ctx, vaa_header.schnorr_key_index, &vaa_header.signature, digest)
  }
}

fn verify_vaa_impl(ctx: Context<VerifyVaa>, raw_vaa: Vec<u8>) -> Result<Vec<u8>> {
  let vaa = VAA::deserialize(&mut raw_vaa.as_slice())?;
  verify(ctx, vaa.header.schnorr_key_index, &vaa.header.signature, vaa.digest()?.to_bytes())?;
  Ok(vaa.body)
}

#[inline(always)]
fn verify(ctx: Context<VerifyVaa>,
  index: u32,
  signature: &VAASchnorrSignature,
  digest: [u8; DIGEST_SIZE]
) -> Result<()> {
  require_eq!(index, ctx.accounts.schnorr_key.index, VerificationV2Error::IndexMismatch);
  ctx.accounts.schnorr_key.schnorr_key.check_signature(&digest, signature)?;
  Ok(())
}
