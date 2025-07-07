use anchor_lang::prelude::{
	AccountInfo,
	AnchorDeserialize,
	AnchorSerialize,
	Clock,
	Discriminator,
	InitSpace,
	Key,
	Program,
	Pubkey,
	Result,
	SolanaSysvar,
	Space,
	System,
	account,
	borsh,
	error_code
};
#[cfg(feature = "idl-build")]
use anchor_lang::{
  IdlBuild,
  idl::types::{
    IdlArrayLen,
    IdlDefinedFields,
    IdlField,
    IdlSerialization,
    IdlType,
    IdlTypeDef,
    IdlTypeDefTy,
  },
};
use anchor_lang::solana_program::{keccak::{Hash, hash}, secp256k1_recover};
use primitive_types::U256;
use std::io::{Read, Write};
use std::ops::{Rem, Shr, Sub};

use crate::hex_literal::hex;
use crate::vaa::VAAThresholdSignature;
use crate::utils::{init_account, SeedPrefix};
use crate::ID;

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ThresholdKey {
  pub key: U256,
}

#[cfg(feature = "idl-build")]
impl IdlBuild for ThresholdKey {
  fn create_type() -> Option<IdlTypeDef> {
    Some(IdlTypeDef {
      name: "ThresholdKey".to_string(),
      docs: vec![],
      serialization: IdlSerialization::Borsh,
      repr: None,
      generics: vec![],
      ty: IdlTypeDefTy::Struct {
        fields: Some(IdlDefinedFields::Named(vec![
          IdlField {
            name: "key".to_string(),
            docs: vec![],
            ty: IdlType::Array(Box::new(IdlType::U8), IdlArrayLen::Value(32)),
          },
        ])),
      },
    })
  }
}

#[error_code]
pub enum ThresholdKeyError {
    #[msg("Signature does not satisfy preconditions")]
    InvalidSignature,
    SignatureVerificationFailed,
}

impl ThresholdKey {
	const HALF_Q: [u8; 32] = hex!("7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF5D576E7357A4501DDFE92F46681B20A0");
	// Using the U256 representation makes verification cheaper
	const Q_U256: [u64; 4] = [
		0xBFD25E8CD0364141,
		0xBAAEDCE6AF48A03B,
		0xFFFFFFFFFFFFFFFE,
		0xFFFFFFFFFFFFFFFF
	];

	pub fn q() -> U256 {
		U256(ThresholdKey::Q_U256)
	}

	pub fn half_q() -> U256 {
		U256::from_big_endian(&ThresholdKey::HALF_Q)
	}

  pub fn px(&self) -> U256 {
    self.key.shr(U256::one())
  }

  pub fn parity(&self) -> bool {
    self.key.bit(0)
  }

	pub fn is_valid(&self) -> bool {
		let px = self.px();
		!px.is_zero() && px.le(&Self::half_q())
	}

	pub fn check_signature(&self, message_hash: &Hash, signature: &VAAThresholdSignature) -> Result<()> {
		let px = self.px();
		let parity = self.parity();
		let q = Self::q();
		let r = signature.r;
		let s = signature.s;

		// Calculate the message challenge
		let mut hash_bytes = [0u8; 85];
		hash_bytes[0..32].copy_from_slice(&px.to_big_endian());
		hash_bytes[32] = parity as u8;
		hash_bytes[33..65].copy_from_slice(&message_hash.to_bytes());
		hash_bytes[65..85].copy_from_slice(&r);

		let e = U256::from_big_endian(&hash(&hash_bytes).to_bytes());

		// Calculate the recovery inputs
		let sp = q.sub(Self::mulmod(px, s, q));
		let ep = Self::mulmod(e, px, q);

		if sp.is_zero() || ep.is_zero() {
			return Err(ThresholdKeyError::InvalidSignature.into());
		}

		// Recover the signer address
		let mut signature_bytes = [0u8; 64];
		// this is r
		signature_bytes[0..32].copy_from_slice(&px.to_big_endian());
		// this is s
		signature_bytes[32..64].copy_from_slice(&ep.to_big_endian());

		let recovered_pubkey = secp256k1_recover::secp256k1_recover(&sp.to_big_endian(), parity as u8, &signature_bytes).unwrap();
		let recovered_address = &hash(&recovered_pubkey.to_bytes()).to_bytes()[12..];

		if recovered_address != r {
			return Err(ThresholdKeyError::SignatureVerificationFailed.into());
		}

		Ok(())
	}

	#[inline(always)]
	fn mulmod(a: U256, b: U256, c: U256) -> U256 {
		let result = a.full_mul(b).rem(c);
		U256(result.0[0..4].try_into().unwrap())
	}
}

impl Space for ThresholdKey {
  const INIT_SPACE: usize = 32;
}

impl AnchorSerialize for ThresholdKey {
	fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
		if !self.is_valid() {
			return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "Invalid threshold key"));
		}

		writer.write_all(&self.key.to_big_endian())?;
		Ok(())
	}
}

impl AnchorDeserialize for ThresholdKey {
	fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
		let mut key_buf = [0u8; 32];
		reader.read_exact(&mut key_buf)?;
		let key = ThresholdKey { key: U256::from_big_endian(&key_buf) };

		Ok(key)
	}
}


#[account]
#[derive(InitSpace)]
pub struct ThresholdKeyAccount {
  pub index: u32,
  pub tss_key: ThresholdKey,
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

impl SeedPrefix for ThresholdKeyAccount {
  const SEED_PREFIX: &'static [u8] = b"thresholdkey";
}

pub fn init_threshold_key_account<'info>(
  new_threshold_key: AccountInfo<'info>,
  tss_index: u32,
  tss_key: ThresholdKey,
  system_program: &Program<'info, System>,
  payer: AccountInfo<'info>
) -> Result<()> {
  // We need to parse the threshold key append VAA payload
  // to perform the derivation.
  // This is why the initialization happens manually here.

  let (pubkey, bump) = Pubkey::find_program_address(
    &[ThresholdKeyAccount::SEED_PREFIX, &tss_index.to_le_bytes()],
    &ID,
  );

  if pubkey != new_threshold_key.key() {
    return Err(AppendThresholdKeyError::AccountMismatchTSSKeyIndex.into());
  }

  let threshold_key_seeds = [ThresholdKeyAccount::SEED_PREFIX, &tss_index.to_le_bytes(), &[bump]];

  init_account(
    new_threshold_key.clone(),
    &threshold_key_seeds,
    &system_program,
    payer,
    ThresholdKeyAccount{
      index: tss_index,
      tss_key,
      expiration_timestamp: 0,
    }
  )?;

  Ok(())
}


#[error_code]
pub enum AppendThresholdKeyError {
  InvalidVAA,
  InvalidGovernanceChainId,
  InvalidGovernanceAddress,
  #[msg("New key must have strictly greater index")]
  InvalidNewKeyIndex,
  #[msg("Old threshold key must be the latest key")]
  InvalidOldThresholdKey,
  #[msg("TSS account pubkey mismatches TSS key index")]
  AccountMismatchTSSKeyIndex,
}

#[test]
fn half_q_is_correct() {
	assert_eq!(ThresholdKey::q().shr(U256::one()), ThresholdKey::half_q());
}

#[test]
fn q_is_correct() {
	let q = U256::from_str_radix("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16).unwrap();
	assert_eq!(ThresholdKey::q(), q);
}