use anchor_lang::prelude::{
  AnchorDeserialize,
  Result,
  borsh::{BorshDeserialize},
};
use anchor_lang::solana_program::keccak::{Hash, hash};
use primitive_types::U256;
use std::io::{Read, Error, ErrorKind};
use byteorder::{BigEndian, ReadBytesExt};

use crate::schnorr_key::SchnorrKey;

pub struct VAASchnorrSignature {
  pub r: [u8; 20],
  pub s: U256,
}

impl VAASchnorrSignature {
  #[inline(always)]
  pub fn is_valid(&self) -> bool {
    !self.s.is_zero() && self.s.lt(&SchnorrKey::q()) && !self.r.iter().all(|r| *r == 0)
  }
}

impl AnchorDeserialize for VAASchnorrSignature {
  #[inline(always)]
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let mut r = [0u8; 20];
    reader.read_exact(&mut r)?;
    let mut s = [0u8; 32];
    reader.read_exact(&mut s)?;
    Ok(Self { r, s: U256::from_big_endian(&s) })
  }
}

pub struct VAAHeader {
  pub schnorr_key_index: u32,
  pub signature: VAASchnorrSignature,
}

impl VAAHeader {
  pub const SIZE: usize = 57;
}

impl AnchorDeserialize for VAAHeader {
  #[inline(always)]
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let version = reader.read_u8()?;
    if version != 2 {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid version"));
    }
    let schnorr_key_index = reader.read_u32::<BigEndian>()?;
    let signature = VAASchnorrSignature::deserialize_reader(reader)?;
    if !signature.is_valid() {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid signature"));
    }
    
    Ok(Self { schnorr_key_index, signature })
  }
}

pub struct VAA {
  pub header: VAAHeader,
  pub body: Vec<u8>,
}

impl VAA {
  const BODY_MIN_SIZE: usize = 51;

  #[inline(always)]
  pub fn digest(&self) -> Result<Hash> {
    Ok(hash(&hash(&self.body).to_bytes()))
  }
}

// We implement Borsh deserialize instead of Anchor equivalents
// because Anchor forces you to provide an `IdlBuild` implementation too and
// we don't need it.
impl BorshDeserialize for VAA {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let header = VAAHeader::deserialize_reader(reader)?;

    let mut body = Vec::new();
    reader.read_to_end(&mut body)?;
    if body.len() < VAA::BODY_MIN_SIZE {
      return Err(Error::new(ErrorKind::InvalidData, "VAA body too short"));
    }

    Ok(Self { header, body })
  }
}

#[cfg(test)]
mod vaa_tests {
  use super::*;
  use crate::hex;

  #[test]
  fn header_deserializes() {
    let mut header_raw = [0u8; VAAHeader::SIZE];
    header_raw[0] = 2;
    header_raw[5..25].copy_from_slice(hex!("636a8688ef4b82e5a121f7c74d821a5b07d695f3").as_slice());
    header_raw[25..57].copy_from_slice(hex!("aa6d485b7d7b536442ea7777127d35af43ac539a491c0d85ee0f635eb7745b29").as_slice());

    let header = VAAHeader::deserialize(&mut header_raw.as_slice());
    assert!(header.is_ok());
  }

  #[test]
  fn header_size_is_correct() {
    let header_raw = [0u8; VAAHeader::SIZE - 1];
    let header = VAAHeader::deserialize(&mut header_raw.as_slice());
    assert!(header.is_err());
  }
}
