use anchor_lang::prelude::{
	AnchorDeserialize,
	AnchorSerialize,
	Result,
	borsh,
	borsh::{BorshDeserialize, BorshSerialize},
};
use anchor_lang::solana_program::keccak::{Hash, hash};
use primitive_types::U256;
use std::io::{Read, Write};
use byteorder::{BigEndian, ReadBytesExt};

use crate::{threshold_key::{ThresholdKey, ThresholdKeyError}, VAAError};

pub struct VAAThresholdSignature {
  pub r: [u8; 20],
  pub s: U256,
}

impl VAAThresholdSignature {
	pub fn is_valid(&self) -> bool {
		!self.s.is_zero() && self.s.lt(&ThresholdKey::q()) && !self.r.iter().all(|r| *r == 0)
	}
}

impl AnchorSerialize for VAAThresholdSignature {
	fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
		writer.write_all(&self.r)?;
		writer.write_all(&self.s.to_big_endian())?;
		Ok(())
	}
}

impl AnchorDeserialize for VAAThresholdSignature {
	fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
		let mut r = [0u8; 20];
		reader.read_exact(&mut r)?;
		let mut s = [0u8; 32];
		reader.read_exact(&mut s)?;
		Ok(Self { r, s: U256::from_big_endian(&s) })
	}
}

pub struct VAAHeader {
	pub version: u8,
	pub tss_index: u32,
	pub signature: VAAThresholdSignature,
}

impl VAAHeader {
	pub fn check_valid(&self) -> Result<()> {
		if self.version != 2 {
			return Err(VAAError::InvalidVersion.into());
		}
		if !self.signature.is_valid() {
			return Err(ThresholdKeyError::InvalidSignature.into());
		}
		Ok(())
	}
}

impl AnchorSerialize for VAAHeader {
	fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
		writer.write_all(&[self.version])?;
		writer.write_all(&self.tss_index.to_be_bytes())?;
		self.signature.serialize(writer)?;
		Ok(())
	}
}

impl AnchorDeserialize for VAAHeader {
	fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
		let version = reader.read_u8()?;
		let tss_index = reader.read_u32::<BigEndian>()?;
		let signature = VAAThresholdSignature::deserialize_reader(reader)?;
		Ok(Self { version, tss_index, signature })
	}
}

pub struct VAAEnvelope {	
  pub timestamp: u32,
  pub nonce: u32,
  pub emitter_chain_id: u16,
  pub emitter_address: [u8; 32],
  pub sequence: u64,
  pub consistency_level: u8,
}

impl AnchorSerialize for VAAEnvelope {
	fn serialize<W: Write>(&self, writer: &mut W) -> std::result::Result<(), std::io::Error> {
		writer.write_all(&self.timestamp.to_be_bytes())?;
		writer.write_all(&self.nonce.to_be_bytes())?;
		writer.write_all(&self.emitter_chain_id.to_be_bytes())?;
		writer.write_all(&self.emitter_address)?;
		writer.write_all(&self.sequence.to_be_bytes())?;
		writer.write_all(&[self.consistency_level])?;
		Ok(())
	}
}

impl AnchorDeserialize for VAAEnvelope {
	fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
		let timestamp = reader.read_u32::<BigEndian>()?;
		let nonce = reader.read_u32::<BigEndian>()?;
		let emitter_chain_id = reader.read_u16::<BigEndian>()?;
		let mut emitter_address = [0u8; 32];
		reader.read_exact(&mut emitter_address)?;
		let sequence = reader.read_u64::<BigEndian>()?;
		let consistency_level = reader.read_u8()?;
		Ok(Self { timestamp, nonce, emitter_chain_id, emitter_address, sequence, consistency_level })
	}
}

#[derive(BorshSerialize)]
pub struct VAA {
	pub header: VAAHeader,
	pub envelope: VAAEnvelope,
	pub payload: Vec<u8>,
}

impl BorshDeserialize for VAA {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::result::Result<Self, std::io::Error> {
    let header = VAAHeader::deserialize_reader(reader)?;
    let envelope = VAAEnvelope::deserialize_reader(reader)?;

    // Read the remaining bytes into `payload`
    let mut payload = Vec::new();
    reader.read_to_end(&mut payload)?;

    Ok(Self {
      header,
      envelope,
      payload,
    })
  }
}

#[cfg(feature = "idl-build")]
impl anchor_lang::IdlBuild for VAA {}

impl VAA {
	pub fn check_valid(&self) -> Result<()> {
		self.header.check_valid()
	}

	pub fn message_hash(&self) -> Result<Hash> {
		let mut body: Vec<u8> = Vec::new();
		self.envelope.serialize(&mut body)?;

		body.extend_from_slice(&self.payload);

		// Single hash
		// Ok(hash(body.as_ref()))
		// Double hash
		Ok(hash(&hash(body.as_ref()).to_bytes()))
	}
}
