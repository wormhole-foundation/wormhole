use anchor_lang::prelude::{AnchorDeserialize, Result, error_code};
use byteorder::{BigEndian, ReadBytesExt};
use std::io::{Cursor, Read};

use crate::schnorr_key::SchnorrKey;
use crate::hex;

pub struct AppendSchnorrKeyMessage {
  pub tss_index: u32,
  pub tss_key: SchnorrKey,
  pub expiration_delay_seconds: u32,
}

#[error_code]
pub enum AppendSchnorrKeyDecodeError {
  InvalidModule,
  InvalidAction,
  InvalidPayload,
}

// Module ID for the VerificationV2 contract, ASCII "TSS"
pub const MODULE_VERIFICATION_V2: [u8; 32] =
  hex!("0000000000000000000000000000000000000000000000000000000000545353");

// Action ID for appending a schnorr key
pub const ACTION_APPEND_THRESHOLD_KEY: u8 = 0x01;

impl AppendSchnorrKeyMessage {
  pub fn deserialize(vaa_payload: &[u8]) -> Result<Self> {
    let mut cursor = Cursor::new(vaa_payload);
    let mut module = [0; 32];
    cursor.read_exact(&mut module)?;
    let action = cursor.read_u8()?;
    let tss_index = cursor.read_u32::<BigEndian>()?;
    let tss_key = SchnorrKey::deserialize_reader(&mut cursor)?;
    let expiration_delay_seconds = cursor.read_u32::<BigEndian>()?;

    // Validate the module and action
    if module != MODULE_VERIFICATION_V2 {
      return Err(AppendSchnorrKeyDecodeError::InvalidModule.into());
    }

    if action != ACTION_APPEND_THRESHOLD_KEY {
      return Err(AppendSchnorrKeyDecodeError::InvalidAction.into());
    }

    // We check that the rest of the VAA is fine but we don't really need the shards here.
    let remaining_bytes = vaa_payload.len() - cursor.position() as usize;
    if remaining_bytes != 32 {
      return Err(AppendSchnorrKeyDecodeError::InvalidPayload.into());
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
