//! Instruction deserialization/handling code

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};

use std::{
    convert::TryInto,
    io::{self, Cursor, Read, Write},
};

use crate::error::{Error, Error::*};

/// Present at the beginning of every EE-VAA instruction
pub const EEVAA_MAGIC: &'static [u8] = b"WHEV"; // Wormhole EE VAA

/// Top-level instruction data type
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum EEVAAInstruction {
    /// Pass an EE-VAA to the bridge
    PostEEVAA(EEVAA),
    /// Pass initialization params to this program
    Initialize(InitParams),
}

/// An enum used to distinguish between instructions in the
/// serialization format. It is best practice to match variant names
/// with variants of [`EEVAAInstruction`].
#[repr(u8)]
pub enum InstructionKind {
    PostEEVAA = 1,
    Initialize = 2,
}

impl EEVAAInstruction {
    /// QoL wrapper for [`Self::deserialize_from_reader`]
    #[inline]
    pub fn deserialize(buf: &[u8]) -> Result<Self, Error> {
        Self::deserialize_from_reader(Cursor::new(buf))
    }

    /// Deserialize the custom Instruction format and underlying data
    pub fn deserialize_from_reader<R: Read>(mut r: R) -> Result<Self, Error> {
        let mut magic = vec![0; EEVAA_MAGIC.len()];

        r.read_exact(&mut magic)
            .map_err(|_| UnexpectedEndOfBuffer)?;

        if magic != EEVAA_MAGIC {
            return Err(Error::InvalidMagic);
        }

        let kind_byte = r.read_u8().map_err(|_| UnexpectedEndOfBuffer)?;

        let i = match kind_byte {
            n if n == InstructionKind::PostEEVAA as u8 => {
                Self::PostEEVAA(EEVAA::deserialize_from_reader(r)?)
            }
            n if n == InstructionKind::Initialize as u8 => {
                Self::Initialize(InitParams::deserialize_from_reader(r)?)
            }
            _other => return Err(InvalidInstructionKind),
        };

        Ok(i)
    }

    /// Turns this instruction into bytes.
    ///
    /// Format:
    ///
    /// | Name             | Length in bytes           | Description                                                     |
    /// |------------------|---------------------------|-----------------------------------------------------------------|
    /// | Magic            | [`EEVAA_MAGIC`] length   | Must match [`EEVAA_MAGIC`] exactly                             |
    /// | Instruction kind | 1                         | Decides [`InstructionKind`] on deserialization                  |
    /// | Payload          | Decided by inner struct   | Each [`EEVAAInstruction`] variant is responsible for its format |
    pub fn serialize(&self) -> Result<Vec<u8>, io::Error> {
        // Start with a copy of the magic
        let mut buf = EEVAA_MAGIC.to_owned();

        use EEVAAInstruction::*;
        match self {
            PostEEVAA(eevaa) => {
                buf.push(InstructionKind::PostEEVAA as u8);
                buf.append(&mut eevaa.serialize()?);
            }
            Initialize(_) => {
                buf.push(InstructionKind::Initialize as u8);
            }
        }

        Ok(buf)
    }
}

/// EE VAA representation
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct EEVAA {
    /// Can be anything, used to distinguish between EEVAAs with
    /// identical payloads and prevent EEVAA account address conflicts
    pub id: u64,
    /// The data to pass along the guardian set gossip network.
    pub payload: Vec<u8>,
}

impl EEVAA {
    /// QoL wrapper for [`Self::deserialize_from_reader`]
    #[inline]
    pub fn deserialize(bytes: &[u8]) -> Result<Self, Error> {
        Self::deserialize_from_reader(Cursor::new(bytes))
    }

    /// Deserialize this EE-VAA
    pub fn deserialize_from_reader(mut r: impl Read) -> Result<Self, Error> {
        // All results boil down to the same type of error, we use a
        // closure to boil it down to a single map_err()
        let mut f = || -> io::Result<Self> {
            let id = r.read_u64::<BigEndian>()?;
            let payload_len = r.read_u16::<BigEndian>()?;

            let mut payload = vec![0; payload_len as usize];

            r.read_exact(payload.as_mut_slice())?;

            Ok(Self { id, payload })
        };

        f().map_err(|_| UnexpectedEndOfBuffer)
    }

    /// Turns this EE VAA into bytes.
    ///
    /// Format:
    ///
    /// | Name   | Length in bytes             | Description                                                                                           |
    /// |--------|-----------------------------|-------------------------------------------------------------------------------------------------------|
    /// | ID     | 8                           | Arbitrary, unique, big endian; used in seeding to prevent account address conflicts for resent EEVAAs |
    /// | Length | 2                           | Big endian, denotes size of the rest of the buffer                                                    |
    /// | Data   | decided by the length field |                                                                                                       |
    pub fn serialize(&self) -> Result<Vec<u8>, io::Error> {
        let mut c = Cursor::new(Vec::new());
        c.write_u64::<BigEndian>(self.id)?;

        c.write_u16::<BigEndian>(
            self.payload
                .len()
                .try_into()
                .map_err(|_| io::Error::new(io::ErrorKind::Other, "Could not write payload len"))?,
        )?;

        c.write_all(self.payload.as_slice())?;

        Ok(c.into_inner())
    }
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct InitParams {
    /// How many lamports to initially put in the EEVAA fee account
    pub eevaa_fee_acc_rent: u64,
}

impl InitParams {
    pub fn deserialize_from_reader(mut r: impl Read) -> Result<Self, Error> {
        let eevaa_fee_acc_rent = r
            .read_u64::<BigEndian>()
            .map_err(|_e| UnexpectedEndOfBuffer)?;

        Ok(Self { eevaa_fee_acc_rent })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    pub type ErrBox = Box<dyn std::error::Error>;

    #[test]
    fn test_serde_eevaa_basic() -> Result<(), ErrBox> {
        let a = EEVAA {
            id: 0xDEADBEEFDEADBABE,
            payload: vec![0x42],
        };

        let buf = a.serialize()?;

        let b = EEVAA::deserialize_from_reader(Cursor::new(buf))?;

        assert_eq!(a, b);

        Ok(())
    }

    #[test]
    fn test_serde_instruction_basic() -> Result<(), ErrBox> {
        let eevaa = EEVAA {
            id: 0xDEADBEEFDEADBABE,
            payload: vec![0x42],
        };
        let i_a = EEVAAInstruction::PostEEVAA(eevaa);

        let buf = i_a.serialize()?;

        let i_b = EEVAAInstruction::deserialize(buf.as_slice())?;

        assert_eq!(i_a, i_b);

        Ok(())
    }
}
