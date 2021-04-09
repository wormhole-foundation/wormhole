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
    pub payload: Option<VAABody>,
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
            payload: None,
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
        v.write(payload_data.as_slice())?;

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

        let mut payload_d = Vec::new();
        rdr.read_to_end(&mut payload_d)?;
        v.payload = Some(VAABody::deserialize(&payload_d)?);

        Ok(v)
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum VAABody {
    UpdateGuardianSet(BodyUpdateGuardianSet),
    Message(BodyMessage),
    UpgradeContract(BodyContractUpgrade),
}

impl VAABody {
    fn action_id(&self) -> u8 {
        match self {
            VAABody::UpdateGuardianSet(_) => 0x01,
            VAABody::UpgradeContract(_) => 0x02,
            VAABody::Message(_) => 0x10,
        }
    }

    fn deserialize(data: &Vec<u8>) -> Result<VAABody, Error> {
        let mut payload_data = Cursor::new(data);
        let action = payload_data.read_u8()?;

        let payload = match action {
            0x01 => {
                VAABody::UpdateGuardianSet(BodyUpdateGuardianSet::deserialize(&mut payload_data)?)
            }
            0x02 => VAABody::UpgradeContract(BodyContractUpgrade::deserialize(&mut payload_data)?),
            0x10 => VAABody::Message(BodyMessage::deserialize(&mut payload_data)?),
            _ => {
                return Err(Error::InvalidVAAAction);
            }
        };

        Ok(payload)
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        match self {
            VAABody::Message(b) => b.serialize(),
            VAABody::UpdateGuardianSet(b) => b.serialize(),
            VAABody::UpgradeContract(b) => b.serialize(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    pub new_index: u32,
    pub new_keys: Vec<[u8; 20]>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyMessage {
    pub emitter_chain: u8,
    pub emitter_address: ForeignAddress,
    pub nonce: u32,
    pub data: Vec<u8>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyContractUpgrade {
    pub chain_id: u8,
    pub buffer: Pubkey,
}

impl BodyContractUpgrade {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyContractUpgrade, Error> {
        let chain_id = data.read_u8()?;
        let mut key: [u8; 32] = [0; 32];
        data.read(&mut key[..])?;

        Ok(BodyContractUpgrade {
            chain_id,
            buffer: Pubkey::new(&key[..]),
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u8(self.chain_id)?;
        v.write(&self.buffer.to_bytes())?;

        Ok(v.into_inner())
    }
}

impl BodyUpdateGuardianSet {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyUpdateGuardianSet, Error> {
        let new_index = data.read_u32::<BigEndian>()?;

        let keys_len = data.read_u8()?;
        let mut keys = Vec::with_capacity(keys_len as usize);
        for _ in 0..keys_len {
            let mut key: [u8; 20] = [0; 20];
            data.read(&mut key)?;
            keys.push(key);
        }

        Ok(BodyUpdateGuardianSet {
            new_index,
            new_keys: keys,
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.new_index)?;
        v.write_u8(self.new_keys.len() as u8)?;

        for k in self.new_keys.iter() {
            v.write(k)?;
        }

        Ok(v.into_inner())
    }
}

impl BodyMessage {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyMessage, Error> {
        let emitter_chain = data.read_u8()?;

        let mut emitter_address: ForeignAddress = ForeignAddress::default();
        data.read_exact(&mut emitter_address)?;

        let nonce = data.read_u32::<BigEndian>()?;

        let mut payload: Vec<u8> = vec![];
        data.read(&mut payload)?;

        Ok(BodyMessage {
            emitter_chain,
            emitter_address,
            nonce,
            data: payload,
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());

        v.write_u8(self.emitter_chain)?;
        v.write(&self.emitter_address)?;
        v.write_u32::<BigEndian>(self.nonce)?;
        v.write(&self.data)?;

        Ok(v.into_inner())
    }
}

#[cfg(test)]
mod tests {
    use hex;

    use crate::vaa::BodyContractUpgrade;
    use crate::vaa::{BodyUpdateGuardianSet, Signature, VAABody, VAA};
    use solana_program::pubkey::Pubkey;

    #[test]
    fn serialize_deserialize_vaa_guardian() {
        let vaa = VAA {
            version: 8,
            guardian_set_index: 3,
            signatures: vec![Signature {
                index: 1,
                r: [2; 32],
                s: [2; 32],
                v: 7,
            }],
            timestamp: 83,
            payload: Some(VAABody::UpdateGuardianSet(BodyUpdateGuardianSet {
                new_index: 29,
                new_keys: vec![],
            })),
        };

        let data = vaa.serialize().unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa)
    }

    #[test]
    fn serialize_deserialize_vaa_contract_upgrade() {
        let vaa = VAA {
            version: 8,
            guardian_set_index: 3,
            signatures: vec![Signature {
                index: 1,
                r: [2; 32],
                s: [2; 32],
                v: 7,
            }],
            timestamp: 83,
            payload: Some(VAABody::UpgradeContract(BodyContractUpgrade {
                chain_id: 3,
                buffer: Pubkey::new_unique(),
            })),
        };

        let data = vaa.serialize().unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa)
    }

    #[test]
    fn parse_given_guardian_set_update() {
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![Signature {
                index: 0,
                r: [
                    51, 130, 199, 26, 76, 121, 225, 81, 138, 108, 226, 156, 145, 86, 159, 100, 39,
                    166, 10, 149, 105, 106, 53, 21, 184, 194, 52, 11, 106, 207, 253, 114,
                ],
                s: [
                    51, 21, 189, 16, 17, 170, 119, 159, 34, 87, 56, 130, 164, 237, 254, 27, 130, 6,
                    84, 142, 19, 72, 113, 162, 63, 139, 160, 193, 199, 208, 181, 237,
                ],
                v: 1,
            }],
            timestamp: 3000,
            payload: Some(VAABody::UpdateGuardianSet(BodyUpdateGuardianSet {
                new_index: 1,
                new_keys: vec![[
                    190, 250, 66, 157, 87, 205, 24, 183, 248, 164, 217, 26, 45, 169, 171, 74, 240,
                    93, 15, 190,
                ]],
            })),
        };
        let data = hex::decode("010000000001003382c71a4c79e1518a6ce29c91569f6427a60a95696a3515b8c2340b6acffd723315bd1011aa779f22573882a4edfe1b8206548e134871a23f8ba0c1c7d0b5ed0100000bb8010000000101befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }

    #[test]
    fn parse_given_contract_upgrade() {
        let vaa = VAA {
            version: 1,
            guardian_set_index: 2,
            signatures: vec![Signature {
                index: 0,
                r: [
                    72, 156, 56, 20, 222, 146, 161, 112, 22, 97, 69, 59, 188, 199, 130, 240, 89,
                    249, 241, 79, 96, 27, 235, 10, 99, 16, 56, 80, 232, 188, 235, 11,
                ],
                s: [
                    65, 19, 144, 42, 104, 122, 52, 0, 126, 7, 43, 127, 120, 85, 5, 21, 216, 207,
                    78, 73, 213, 207, 142, 103, 211, 192, 100, 90, 27, 98, 176, 98,
                ],
                v: 1,
            }],
            timestamp: 4000,
            payload: Some(VAABody::UpgradeContract(BodyContractUpgrade {
                chain_id: 2,
                buffer: Pubkey::new(&[
                    146, 115, 122, 21, 4, 243, 179, 223, 140, 147, 203, 133, 198, 74, 72, 96, 187,
                    39, 14, 38, 2, 107, 110, 55, 240, 149, 53, 106, 64, 111, 106, 244,
                ]),
            })),
        };
        let data = hex::decode("01000000020100489c3814de92a1701661453bbcc782f059f9f14f601beb0a63103850e8bceb0b4113902a687a34007e072b7f78550515d8cf4e49d5cf8e67d3c0645a1b62b0620100000fa0020292737a1504f3b3df8c93cb85c64a4860bb270e26026b6e37f095356a406f6af4").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }
}
