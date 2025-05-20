use anchor_lang::prelude::*;
use primitive_types::U256;
use std::io::{Cursor, Read, Write};
use std::ops::{Rem, Shr, Sub};
use anchor_lang::solana_program::keccak::{Hash, hash};
use libsecp256k1::{Message, RecoveryId, Signature, recover};
use crate::vaa::VAAThresholdSignature;

#[derive(Clone, Debug)]
pub struct ThresholdKey {
  pub key: U256,
}

#[cfg(feature = "idl-build")]
impl anchor_lang::IdlBuild for ThresholdKey {}

#[error_code]
pub enum ThresholdKeyError {
    #[msg("Invalid scalar value")]
    InvalidScalar,
    #[msg("Invalid signature")]
    InvalidSignature,
}

impl ThresholdKey {
	pub fn q() -> U256 {
		// TODO: Move this to a constant
		U256::from_str_radix("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16).unwrap()
	}

	pub fn half_q() -> U256 {
		// TODO: Move this to a constant
		Self::q().shr(U256::one())
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
		let mut hash_bytes = Vec::new();
		hash_bytes.extend_from_slice(&px.to_big_endian());
		hash_bytes.push(parity as u8);
		hash_bytes.extend_from_slice(&message_hash.to_bytes());
		hash_bytes.extend_from_slice(&r);

		let e = U256::from_big_endian(&hash(&hash_bytes).to_bytes());

		// Calculate the recovery inputs
		// FIXME: Is the overflow correct on the sp/ep calculations?
		let sp = q.sub(Self::mulmod(px, s, q));
		let ep = Self::mulmod(e, px, q);

		if sp.is_zero() || ep.is_zero() {
			return Err(ThresholdKeyError::InvalidSignature.into());
		}

		// Recover the signer address
		let mut signature_bytes = [0u8; 64];
		signature_bytes[0..32].copy_from_slice(&px.to_big_endian());
		signature_bytes[32..64].copy_from_slice(&ep.to_big_endian());

		let message = Message::parse_slice(&sp.to_big_endian()).unwrap();
		let signature = Signature::parse_standard(&signature_bytes).unwrap();
		let recovery_id = RecoveryId::parse(parity as u8).unwrap();
		let recovered = recover(&message, &signature, &recovery_id).unwrap();
		let recovered_address = &hash(&recovered.serialize()[1..]).to_bytes()[12..];

		if recovered_address != r {
			return Err(ThresholdKeyError::InvalidSignature.into());
		}

		Ok(())
	}

	fn mulmod(a: U256, b: U256, c: U256) -> U256 {
		let result = a.full_mul(b).rem(c);
		U256(result.0[0..3].try_into().unwrap())
	}
}

impl Space for ThresholdKey {
  const INIT_SPACE: usize = 32;
}

impl AnchorSerialize for ThresholdKey {
	fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
		writer.write_all(&self.key.to_big_endian())?;
		Ok(())
	}
}

impl AnchorDeserialize for ThresholdKey {
	fn deserialize(data: &mut &[u8]) -> std::result::Result<Self, std::io::Error> {
		Self::deserialize_reader(&mut Cursor::new(data))
	}

	fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
		let mut key = [0u8; 32];
		reader.read_exact(&mut key)?;
		let key = ThresholdKey { key: U256::from_big_endian(&key) };

		if !key.is_valid() {
			return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "Invalid threshold key"));
		}

		Ok(key)
	}
}
