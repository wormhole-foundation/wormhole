use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use std::io::{Cursor, Read, Write};
use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::hash;

pub struct VAA {
  // Header
  pub version: u8,
  pub guardian_set_index: u32,
  pub signature: [u8; 64],
	pub recovery_id: u8,

  // Body
  pub timestamp: u64,
  pub nonce: u64,
  pub emitter_chain_id: u16,
  pub emitter_address: [u8; 32],
  pub sequence: u64,
  pub consistency_level: u8,
  pub payload: Vec<u8>,
}

impl VAA {
	pub const HEADER_SIZE: usize = 1 + 4 + 64 + 1;
	pub const BODY_SIZE: usize = 8 + 8 + 2 + 32 + 8 + 1;
	pub const PAYLOAD_OFFSET: usize = Self::HEADER_SIZE + Self::BODY_SIZE;

	pub fn get_seed_bytes(encoded_vaa: &[u8]) -> &[u8] {
		// Extract the index from the encoded VAA
		const INDEX_OFFSET: usize = 1;
		const INDEX_SIZE: usize = (u32::BITS / u8::BITS) as usize;
		&encoded_vaa[INDEX_OFFSET..INDEX_OFFSET + INDEX_SIZE]
	}

	pub fn serialize(&self, data: &mut [u8]) -> Result<()> {
		let mut cursor = Cursor::new(data);

		cursor.write_u8(self.version)?;
		cursor.write_u32::<BigEndian>(self.guardian_set_index)?;
		cursor.write_all(&self.signature)?;
		cursor.write_u8(self.recovery_id)?;
		cursor.write_u64::<BigEndian>(self.timestamp)?;
		cursor.write_u64::<BigEndian>(self.nonce)?;
		cursor.write_u16::<BigEndian>(self.emitter_chain_id)?;
		cursor.write_all(&self.emitter_address)?;
		cursor.write_u64::<BigEndian>(self.sequence)?;
		cursor.write_u8(self.consistency_level)?;
		cursor.write_all(&self.payload)?;

		Ok(())
	}

	pub fn deserialize(data: &[u8]) -> Result<(Self, [u8; 32])> {
		let mut cursor = Cursor::new(data);

		let version = cursor.read_u8()?;
		let guardian_set_index = cursor.read_u32::<BigEndian>()?;
		let mut signature = [0; 64];
		cursor.read_exact(&mut signature)?;
		let recovery_id = cursor.read_u8()?;

		let body_start = cursor.position() as usize;
		
		let timestamp = cursor.read_u64::<BigEndian>()?;
		let nonce = cursor.read_u64::<BigEndian>()?;
		let emitter_chain_id = cursor.read_u16::<BigEndian>()?;
		let mut emitter_address = [0; 32];
		cursor.read_exact(&mut emitter_address)?;

		let sequence = cursor.read_u64::<BigEndian>()?;
		let consistency_level = cursor.read_u8()?;
		let mut payload = Vec::new();
		cursor.read_to_end(&mut payload)?;

		let double_hash = hash(&hash(&data[body_start..]).to_bytes()).to_bytes();

		Ok(
			(
				Self {
					version,
					guardian_set_index,
					signature,
					recovery_id,
					timestamp,
					nonce,
					emitter_chain_id,
					emitter_address,
					sequence,
					consistency_level,
					payload,
				},
				double_hash,
			)
		)
	}
}
