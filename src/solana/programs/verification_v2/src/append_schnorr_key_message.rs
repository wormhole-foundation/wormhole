use anchor_lang::prelude::{AnchorSerialize, AnchorDeserialize};
use byteorder::{BigEndian, ReadBytesExt};
use std::io::{Write, Read, Error, ErrorKind};

use crate::schnorr_key::SchnorrKey;
use crate::hex;

#[derive(Clone)]
pub struct AppendSchnorrKeyMessage {
  pub schnorr_key_index: u32,
  pub schnorr_key: SchnorrKey,
  pub expiration_delay_seconds: u32,
}

// Module ID for the VerificationV2 contract, ASCII "TSS"
pub const MODULE_VERIFICATION_V2: [u8; 32] =
  hex!("0000000000000000000000000000000000000000000000000000000000545353");

// Action ID for appending a schnorr key
pub const ACTION_APPEND_SCHNORR_KEY: u8 = 0x01;

impl AnchorSerialize for AppendSchnorrKeyMessage {
  fn serialize<W: Write>(&self, _writer: &mut W) -> std::io::Result<()> {
    panic!("Deliberately not implemented, but trait is required by PostedVaa's generic parameter");
  }
}

impl AnchorDeserialize for AppendSchnorrKeyMessage {
  fn deserialize_reader<R: Read>(reader: &mut R) -> std::io::Result<Self> {
    let mut module = [0; 32];
    reader.read_exact(&mut module)?;
    let action = reader.read_u8()?;

    let schnorr_key_index = reader.read_u32::<BigEndian>()?;
    let schnorr_key = SchnorrKey::deserialize_reader(reader)?;
    let expiration_delay_seconds = reader.read_u32::<BigEndian>()?;

    // Validate the module and action
    if module != MODULE_VERIFICATION_V2 {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid module"));
    }

    if action != ACTION_APPEND_SCHNORR_KEY {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid action"));
    }

    // We check that the rest of the VAA is fine but we don't really need the shards here.
    let mut remaining_bytes = [0; 32];
    if reader.read_exact(&mut remaining_bytes).is_err() {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid payload"));
    }

    if reader.read_u8().is_ok() {
      return Err(Error::new(ErrorKind::InvalidData, "Invalid payload"));
    }

    Ok(Self {
      schnorr_key_index,
      schnorr_key,
      expiration_delay_seconds,
    })
  }
}
