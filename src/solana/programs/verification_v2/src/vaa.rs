use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::{Hash, hash};
use primitive_types::U256;
use borsh::{BorshSerialize, BorshDeserialize};
use std::io::{Cursor, Read, Write};
use byteorder::{BigEndian, ReadBytesExt};

use crate::threshold_key::ThresholdKey;

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
	fn deserialize(data: &mut &[u8]) -> std::result::Result<Self, std::io::Error> {
		Self::deserialize_reader(&mut Cursor::new(data))
	}

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
	pub fn is_valid(&self) -> bool {
		self.version == 2 && self.signature.is_valid()
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
	fn deserialize(data: &mut &[u8]) -> std::result::Result<Self, std::io::Error> {
		Self::deserialize_reader(&mut Cursor::new(data))
	}

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
	fn deserialize(data: &mut &[u8]) -> std::result::Result<Self, std::io::Error> {
		Self::deserialize_reader(&mut Cursor::new(data))
	}

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

#[derive(BorshSerialize, BorshDeserialize)]
pub struct VAA {
	pub header: VAAHeader,
  pub envelope: VAAEnvelope,
	pub payload: Vec<u8>,
}

#[cfg(feature = "idl-build")]
impl anchor_lang::IdlBuild for VAA {}

impl VAA {
	pub fn is_valid(&self) -> bool {
		self.header.is_valid()
	}

	pub fn message_hash(&self) -> Result<Hash> {
		let mut cursor = Cursor::new(Vec::new());
		self.envelope.serialize(&mut cursor)?;
		cursor.write_all(&self.payload)?;
		Ok(hash(&cursor.get_ref()))
	}
}
