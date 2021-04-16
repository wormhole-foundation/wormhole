use std::io::{Cursor, Read, Write};

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use sha3::Digest;
use solana_program::program_error::ProgramError;

use crate::error::Error;
use solana_program::pubkey::Pubkey;

pub type ForeignAddress = [u8; 32];

#[derive(Clone, Debug, Default, PartialEq)]
pub struct VAA {
    // Header part
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,

    // Body part
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u8,
    pub emitter_address: ForeignAddress,
    pub payload: Vec<u8>,
}

#[derive(Clone, Copy, Debug, Default, PartialEq)]
pub struct Signature {
    pub index: u8,
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
}

impl VAA {
    pub fn new() -> VAA {
        return VAA {
            version: 0,
            guardian_set_index: 0,
            signatures: vec![],
            timestamp: 0,
            emitter_chain: 0,
            emitter_address: [0u8; 32],
            nonce: 0,
            payload: vec![],
        };
    }

    pub fn body_hash(&self) -> Result<[u8; 32], ProgramError> {
        let body = match self.signature_body() {
            Ok(v) => v,
            Err(_) => {
                return Err(ProgramError::InvalidArgument);
            }
        };

        let mut h = sha3::Keccak256::default();
        if let Err(_) = h.write(body.as_slice()) {
            return Err(ProgramError::InvalidArgument);
        };
        Ok(h.finalize().into())
    }

    pub fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v = Cursor::new(Vec::new());

        v.write_u8(self.version)?;
        v.write_u32::<BigEndian>(self.guardian_set_index)?;

        v.write_u8(self.signatures.len() as u8)?;
        for s in self.signatures.iter() {
            v.write_u8(s.index)?;
            v.write(&s.r)?;
            v.write(&s.s)?;
            v.write_u8(s.v)?;
        }

        v.write_u32::<BigEndian>(self.timestamp)?;
        v.write_u32::<BigEndian>(self.nonce)?;
        v.write_u8(self.emitter_chain)?;
        v.write(&self.emitter_address)?;
        v.write(&self.payload)?;

        Ok(v.into_inner())
    }

    pub fn signature_body(&self) -> Result<Vec<u8>, Error> {
        let mut v = Cursor::new(Vec::new());

        v.write_u32::<BigEndian>(self.timestamp)?;
        v.write_u32::<BigEndian>(self.nonce)?;
        v.write_u8(self.emitter_chain)?;
        v.write(&self.emitter_address)?;
        v.write(&self.payload)?;

        Ok(v.into_inner())
    }

    pub fn deserialize(data: &[u8]) -> Result<VAA, Error> {
        let mut rdr = Cursor::new(data);
        let mut v = VAA::new();

        v.version = rdr.read_u8()?;
        v.guardian_set_index = rdr.read_u32::<BigEndian>()?;

        let len_sig = rdr.read_u8()?;
        let mut sigs: Vec<Signature> = Vec::with_capacity(len_sig as usize);
        for _i in 0..len_sig {
            let mut sig = Signature::default();

            sig.index = rdr.read_u8()?;
            rdr.read_exact(&mut sig.r)?;
            rdr.read_exact(&mut sig.s)?;
            sig.v = rdr.read_u8()?;

            sigs.push(sig);
        }
        v.signatures = sigs;

        v.timestamp = rdr.read_u32::<BigEndian>()?;
        v.nonce = rdr.read_u32::<BigEndian>()?;
        v.emitter_chain = rdr.read_u8()?;
        rdr.read_exact(&mut v.emitter_address)?;
        rdr.read_to_end(&mut v.payload)?;

        Ok(v)
    }
}
