use std::any::Any;
use std::io::{Cursor, Read, Write};
use std::mem::size_of;

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use solana_sdk::program_error::ProgramError;

use crate::error::Error;
use crate::error::Error::InvalidVAAFormat;
use crate::instruction::unpack;
use crate::vaa::VAABody::{Undefined, UpdateGuardianSet};

pub type ForeignAddress = [u8; 32];

pub struct VAA {
    // Header part
    version: u8,
    guardian_set_index: u32,
    signature: [u8; 52],

    // Body part
    timestamp: u32,
    payload: VAABody,
}

impl VAA {
    fn new() -> VAA {
        return VAA {
            version: 0,
            guardian_set_index: 0,
            signature: [0; 52],
            timestamp: 0,
            payload: Undefined(),
        };
    }

    fn serialize(&self) -> Vec<u8> {
        let mut v = Cursor::new(Vec::new());

        v.write_u8(self.version);
        v.write_u32::<BigEndian>(self.guardian_set_index);
        v.write(self.signature.as_ref());
        v.write_u32::<BigEndian>(self.timestamp);
        v.write_u8(self.payload.action_id());

        let payload_data = self.payload.serialize();
        v.write_u8(payload_data.len() as u8);
        v.write(payload_data.as_slice());

        v.into_inner()
    }

    fn deserialize(data: &Vec<u8>) -> Result<VAA, std::io::Error> {
        let mut rdr = Cursor::new(data.as_slice());
        let mut v = VAA::new();

        v.version = rdr.read_u8()?;
        v.guardian_set_index = rdr.read_u32::<BigEndian>()?;

        let mut sig: [u8; 52] = [0; 52];
        rdr.read_exact(&mut sig)?;
        v.signature = sig;

        v.timestamp = rdr.read_u32::<BigEndian>()?;

        let mut payload_d = Vec::new();
        rdr.read(&mut payload_d)?;
        v.payload = VAABody::deserialize(&payload_d)?;

        Ok(v)
    }
}

pub enum VAABody {
    Undefined(),
    UpdateGuardianSet(BodyUpdateGuardianSet),
    Transfer(BodyTransfer),
}

impl VAABody {
    fn action_id(&self) -> u8 {
        match self {
            VAABody::Undefined() => { panic!("undefined action") }
            VAABody::UpdateGuardianSet(_) => 0x01,
            VAABody::Transfer(_) => 0x10,
        }
    }

    fn deserialize(data: &Vec<u8>) -> Result<VAABody, std::io::Error> {
        let mut payload_data = Cursor::new(data);
        let action = payload_data.read_u8()?;

        let payload = match action {
            0x01 => {
                let guardian_set_index = payload_data.read_u32::<BigEndian>()?;
                let mut key: [u8; 32] = [0; 32];
                payload_data.read(&mut key)?;

                UpdateGuardianSet(BodyUpdateGuardianSet {
                    new_index: guardian_set_index,
                    new_key: key,
                })
            }
            0x10 => {
                VAABody::Transfer(BodyTransfer::deserialize(&mut payload_data)?)
            }
            _ => {
                Undefined()
            }
        };

        Ok(payload)
    }

    fn serialize(&self) -> Vec<u8> {
        match self {
            VAABody::Transfer(b) => {
                b.serialize()
            }
            VAABody::UpdateGuardianSet(b) => {
                b.serialize()
            }
            VAABody::Undefined() => {
                panic!("undefined action")
            }
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    new_index: u32,
    new_key: [u8; 32],
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyTransfer {
    source_chain: u8,
    target_chain: u8,
    target_address: ForeignAddress,
    token_chain: u8,
    token_address: ForeignAddress,
    amount: u64,
}

impl BodyUpdateGuardianSet {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyUpdateGuardianSet, std::io::Error> {
        let new_index = data.read_u32::<BigEndian>()?;
        let mut new_key: [u8; 32] = [0; 32];
        data.read(&mut new_key)?;

        Ok(BodyUpdateGuardianSet {
            new_index,
            new_key,
        })
    }

    fn serialize(&self) -> Vec<u8> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.new_index);
        v.write(&self.new_key);

        v.into_inner()
    }
}

impl BodyTransfer {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyTransfer, std::io::Error> {
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

    fn serialize(&self) -> Vec<u8> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u8(self.source_chain);
        v.write_u8(self.target_chain);
        v.write(&self.target_address);
        v.write_u8(self.token_chain);
        v.write(&self.token_address);
        v.write_u64::<BigEndian>(self.amount);

        v.into_inner()
    }
}
