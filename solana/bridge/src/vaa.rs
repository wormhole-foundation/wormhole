use std::any::Any;
use std::io::{Cursor, Read, Write};
use std::mem::size_of;

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use sha3::Digest;
use solana_sdk::program_error::ProgramError;

use crate::error::Error;
use crate::error::Error::InvalidVAAFormat;
use crate::instruction::unpack;
use crate::syscalls::{RawKey, SchnorrifyInput, sol_verify_schnorr};
use crate::vaa::VAABody::UpdateGuardianSet;

pub type ForeignAddress = [u8; 32];

#[derive(Clone, Debug, Default, PartialEq)]
pub struct VAA {
    // Header part
    pub version: u8,
    pub guardian_set_index: u32,
    pub signature_sig: [u8; 32],
    pub signature_addr: [u8; 20],

    // Body part
    pub timestamp: u32,
    pub payload: Option<VAABody>,
}

impl VAA {
    pub fn new() -> VAA {
        return VAA {
            version: 0,
            guardian_set_index: 0,
            signature_sig: [0; 32],
            signature_addr: [0; 20],
            timestamp: 0,
            payload: None,
        };
    }

    pub fn verify(&self, guardian_key: &RawKey) -> bool {
        let body = match self.signature_body() {
            Ok(v) => { v }
            Err(_) => { return false; }
        };

        let mut h = sha3::Keccak256::default();
        if let Err(_) = h.write(body.as_slice()) { return false; };
        let hash = h.finalize().into();

        let schnorr_input = SchnorrifyInput::new(*guardian_key, hash,
                                                 self.signature_sig, self.signature_addr);
        sol_verify_schnorr(&schnorr_input)
    }

    pub fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v = Cursor::new(Vec::new());

        v.write_u8(self.version)?;
        v.write_u32::<BigEndian>(self.guardian_set_index)?;
        v.write(self.signature_sig.as_ref())?;
        v.write(self.signature_addr.as_ref())?;
        v.write_u32::<BigEndian>(self.timestamp)?;

        let payload = self.payload.as_ref().ok_or(Error::InvalidVAAAction)?;
        v.write_u8(payload.action_id())?;

        let payload_data = payload.serialize()?;
        v.write(payload_data.as_slice())?;

        Ok(v.into_inner())
    }

    pub fn signature_body(&self) -> Result<Vec<u8>, Error> {
        let mut v = Cursor::new(Vec::new());

        v.write_u32::<BigEndian>(self.timestamp)?;

        let payload = self.payload.as_ref().ok_or(Error::InvalidVAAAction)?;
        v.write_u8(payload.action_id())?;

        let payload_data = payload.serialize()?;
        v.write_u8(payload_data.len() as u8)?;
        v.write(payload_data.as_slice())?;

        Ok(v.into_inner())
    }

    pub fn deserialize(data: &[u8]) -> Result<VAA, Error> {
        let mut rdr = Cursor::new(data);
        let mut v = VAA::new();

        v.version = rdr.read_u8()?;
        v.guardian_set_index = rdr.read_u32::<BigEndian>()?;
        rdr.read_exact(&mut v.signature_sig)?;
        rdr.read_exact(&mut v.signature_addr)?;

        v.timestamp = rdr.read_u32::<BigEndian>()?;

        let mut payload_d = Vec::new();
        rdr.read_to_end(&mut payload_d)?;
        v.payload = Some(VAABody::deserialize(&payload_d)?);

        Ok(v)
    }
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum VAABody {
    UpdateGuardianSet(BodyUpdateGuardianSet),
    Transfer(BodyTransfer),
}

impl VAABody {
    fn action_id(&self) -> u8 {
        match self {
            VAABody::UpdateGuardianSet(_) => 0x01,
            VAABody::Transfer(_) => 0x10,
        }
    }

    fn deserialize(data: &Vec<u8>) -> Result<VAABody, Error> {
        let mut payload_data = Cursor::new(data);
        let action = payload_data.read_u8()?;

        let payload = match action {
            0x01 => {
                VAABody::UpdateGuardianSet(BodyUpdateGuardianSet::deserialize(&mut payload_data)?)
            }
            0x10 => {
                VAABody::Transfer(BodyTransfer::deserialize(&mut payload_data)?)
            }
            _ => {
                return Err(Error::InvalidVAAAction);
            }
        };

        Ok(payload)
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        match self {
            VAABody::Transfer(b) => {
                b.serialize()
            }
            VAABody::UpdateGuardianSet(b) => {
                b.serialize()
            }
        }
    }
}

#[derive(Copy, Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    pub new_index: u32,
    pub new_key: RawKey,
}

#[derive(Copy, Clone, Debug, PartialEq)]
pub struct BodyTransfer {
    pub source_chain: u8,
    pub target_chain: u8,
    pub target_address: ForeignAddress,
    pub token_chain: u8,
    pub token_address: ForeignAddress,
    pub amount: u64,
}

impl BodyUpdateGuardianSet {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyUpdateGuardianSet, Error> {
        let new_index = data.read_u32::<BigEndian>()?;
        let mut new_key_x: [u8; 32] = [0; 32];
        data.read(&mut new_key_x)?;
        let mut new_key_y: [u8; 32] = [0; 32];
        data.read(&mut new_key_y)?;

        Ok(BodyUpdateGuardianSet {
            new_index,
            new_key: RawKey {
                x: new_key_x,
                y: new_key_y,
            },
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.new_index)?;
        v.write(&self.new_key.x)?;
        v.write(&self.new_key.y)?;

        Ok(v.into_inner())
    }
}

impl BodyTransfer {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyTransfer, Error> {
        let source_chain = data.read_u8()?;
        let target_chain = data.read_u8()?;
        let mut target_address: ForeignAddress = ForeignAddress::default();
        data.read(&mut target_address)?;
        let token_chain = data.read_u8()?;
        let mut token_address: ForeignAddress = ForeignAddress::default();
        data.read(&mut token_address)?;
        let amount = data.read_u64::<BigEndian>()?;

        Ok(BodyTransfer {
            source_chain,
            target_chain,
            target_address,
            token_chain,
            token_address,
            amount,
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u8(self.source_chain)?;
        v.write_u8(self.target_chain)?;
        v.write(&self.target_address)?;
        v.write_u8(self.token_chain)?;
        v.write(&self.token_address)?;
        v.write_u64::<BigEndian>(self.amount)?;

        Ok(v.into_inner())
    }
}

#[cfg(test)]
mod tests {
    use std::io::Write;

    use hex;

    use crate::error::Error;
    use crate::syscalls::RawKey;
    use crate::vaa::{BodyTransfer, BodyUpdateGuardianSet, VAA, VAABody};

    #[test]
    fn serialize_deserialize_vaa_transfer() {
        let vaa = VAA {
            version: 8,
            guardian_set_index: 3,
            signature_sig: [7; 32],
            signature_addr: [9; 20],
            timestamp: 83,
            payload: Some(VAABody::Transfer(BodyTransfer {
                source_chain: 1,
                target_chain: 2,
                target_address: [1; 32],
                token_chain: 3,
                token_address: [8; 32],
                amount: 4,
            })),
        };

        let data = vaa.serialize().unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa)
    }

    #[test]
    fn serialize_deserialize_vaa_guardian() {
        let vaa = VAA {
            version: 8,
            guardian_set_index: 3,
            signature_sig: [7; 32],
            signature_addr: [9; 20],
            timestamp: 83,
            payload: Some(VAABody::UpdateGuardianSet(BodyUpdateGuardianSet {
                new_index: 29,
                new_key: RawKey {
                    x: [2; 32],
                    y: [3; 32],
                },
            })),
        };

        let data = vaa.serialize().unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa)
    }
}
