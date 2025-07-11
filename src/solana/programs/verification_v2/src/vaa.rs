use anchor_lang::prelude::{
  AnchorDeserialize,
  AnchorSerialize,
  Result,
  borsh::{BorshDeserialize, BorshSerialize},
};
use anchor_lang::solana_program::keccak::{Hash, hash};
use primitive_types::U256;
use std::io::{Read, Write};
use byteorder::{BigEndian, ReadBytesExt};

use crate::{schnorr_key::{SchnorrKey, SchnorrKeyError}, VAAError};

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

impl AnchorSerialize for VAASchnorrSignature {
  fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
    writer.write_all(&self.r)?;
    writer.write_all(&self.s.to_big_endian())?;
    Ok(())
  }
}

impl AnchorDeserialize for VAASchnorrSignature {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let mut r = [0u8; 20];
    reader.read_exact(&mut r)?;
    let mut s = [0u8; 32];
    reader.read_exact(&mut s)?;
    Ok(Self { r, s: U256::from_big_endian(&s) })
  }
}

pub struct VAAHeader {
  pub version: u8,
  pub schnorr_key_index: u32,
  pub signature: VAASchnorrSignature,
}

impl VAAHeader {
  #[inline(always)]
  pub fn check_valid(&self) -> Result<()> {
    if self.version != 2 {
      return Err(VAAError::InvalidVersion.into());
    }
    if !self.signature.is_valid() {
      return Err(SchnorrKeyError::InvalidSignature.into());
    }
    Ok(())
  }
}

impl AnchorSerialize for VAAHeader {
  fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
    writer.write_all(&[self.version])?;
    writer.write_all(&self.schnorr_key_index.to_be_bytes())?;
    self.signature.serialize(writer)?;
    Ok(())
  }
}

impl AnchorDeserialize for VAAHeader {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let version = reader.read_u8()?;
    let schnorr_key_index = reader.read_u32::<BigEndian>()?;
    let signature = VAASchnorrSignature::deserialize_reader(reader)?;
    Ok(Self { version, schnorr_key_index, signature })
  }
}

pub struct VAA {
  pub header: VAAHeader,
  pub body: Vec<u8>,
}

impl BorshSerialize for VAA {
  fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
      self.header.serialize(writer)?;
      writer.write_all(&self.body)?;
      Ok(())
  }
}

impl BorshDeserialize for VAA {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let header = VAAHeader::deserialize_reader(reader)?;

    let mut body = Vec::new();
    reader.read_to_end(&mut body)?;

    Ok(Self {
      header,
      body,
    })
  }
}

impl VAA {
  pub fn check_valid(&self) -> Result<()> {
    self.header.check_valid()?;
    // See VAABody for necessary fields.
    if self.body.len() < 51 {
      return Err(VAAError::BodyTooSmall.into());
    }
    Ok(())
  }

  #[inline(always)]
  pub fn message_hash(&self) -> Result<Hash> {
    Ok(hash(&hash(&self.body).to_bytes()))
  }
}


pub struct VAABody {  
  pub timestamp: u32,
  pub nonce: u32,
  pub emitter_chain_id: u16,
  pub emitter_address: [u8; 32],
  pub sequence: u64,
  pub consistency_level: u8,
  pub payload: Vec<u8>,
}

// TODO: define the type for the body IDL?
#[cfg(feature = "idl-build")]
impl anchor_lang::IdlBuild for VAABody {}

impl AnchorSerialize for VAABody {
  fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
    writer.write_all(&self.timestamp.to_be_bytes())?;
    writer.write_all(&self.nonce.to_be_bytes())?;
    writer.write_all(&self.emitter_chain_id.to_be_bytes())?;
    writer.write_all(&self.emitter_address)?;
    writer.write_all(&self.sequence.to_be_bytes())?;
    writer.write_all(&[self.consistency_level])?;
    writer.write_all(&self.payload)?;
    Ok(())
  }
}

impl AnchorDeserialize for VAABody {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let timestamp = reader.read_u32::<BigEndian>()?;
    let nonce = reader.read_u32::<BigEndian>()?;
    let emitter_chain_id = reader.read_u16::<BigEndian>()?;
    let mut emitter_address = [0u8; 32];
    reader.read_exact(&mut emitter_address)?;
    let sequence = reader.read_u64::<BigEndian>()?;
    let consistency_level = reader.read_u8()?;
    let mut payload = Vec::new();
    reader.read_to_end(&mut payload)?;
    Ok(Self {
      timestamp,
      nonce,
      emitter_chain_id,
      emitter_address,
      sequence,
      consistency_level,
      payload
    })
  }
}
