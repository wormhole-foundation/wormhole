use byteorder::{BigEndian, ReadBytesExt};
use std::io::{Cursor, Read};
use anchor_lang::prelude::*;

pub struct AppendThresholdKeyMessage {
	pub tss_index: u32,
	pub tss_key: [u8; 20],
	pub expiration_delay_seconds: u32,
}

#[error_code]
pub enum AppendThresholdKeyDecodeError {
	InvalidModule,
	InvalidAction,
}

// Module ID for the VerificationV2 contract, ASCII "TSS"
pub const MODULE_VERIFICATION_V2: [u8; 32] = [
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x53, 0x53,
];

// Action ID for appending a threshold key
pub const ACTION_APPEND_THRESHOLD_KEY: u8 = 0x01;

impl AppendThresholdKeyMessage {
	pub fn deserialize(vaa_body: &[u8]) -> Result<Self> {
		// Decode the VAA body
		let mut cursor = Cursor::new(vaa_body);
		let mut module = [0; 32];
		cursor.read_exact(&mut module)?;
		let action = cursor.read_u8()?;
		let tss_index = cursor.read_u32::<BigEndian>()?;
		let mut tss_key = [0u8; 20];
		cursor.read_exact(&mut tss_key)?;
		let expiration_delay_seconds = cursor.read_u32::<BigEndian>()?;

		// Validate the module and action
		if module != MODULE_VERIFICATION_V2 {
			return Err(AppendThresholdKeyDecodeError::InvalidModule.into());
		}

		if action != ACTION_APPEND_THRESHOLD_KEY {
			return Err(AppendThresholdKeyDecodeError::InvalidAction.into());
		}

		Ok(
			Self {
				tss_index,
				tss_key,
				expiration_delay_seconds,
			}
		)
	}
}
