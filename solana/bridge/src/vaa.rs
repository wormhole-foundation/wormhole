use std::io::{Cursor, Read, Write};

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use primitive_types::U256;
use sha3::Digest;

use crate::error::Error;
use crate::error::Error::InvalidVAAFormat;
use crate::state::AssetMeta;
use crate::syscalls::{sol_syscall_ecrecover, EcrecoverInput, EcrecoverOutput};

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

    pub fn verify(&self, guardian_keys: &[[u8; 20]]) -> bool {
        let body = match self.signature_body() {
            Ok(v) => v,
            Err(_) => {
                return false;
            }
        };

        let mut h = sha3::Keccak256::default();
        if let Err(_) = h.write(body.as_slice()) {
            return false;
        };
        let hash = h.finalize().into();

        for sig in self.signatures.iter() {
            let ecrecover_input = EcrecoverInput::new(sig.r, sig.s, sig.v, hash);
            let res = match sol_syscall_ecrecover(&ecrecover_input) {
                Ok(v) => v,
                Err(_) => {
                    return false;
                }
            };

            if sig.index >= guardian_keys.len() as u8 {
                return false;
            }
            if res.address != guardian_keys[sig.index as usize] {
                return false;
            }
        }

        true
    }

    pub fn body_hash(&self) -> Result<[u8; 32], Error> {
        let body_bytes = self.signature_body()?;

        let mut k = sha3::Keccak256::default();
        if let Err(_) = k.write(body_bytes.as_slice()) {
            return Err(Error::ParseFailed.into());
        };
        let hash = k.finalize();

        return Ok(hash.into());
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
        v.write_u8(payload_data.len() as u8)?;
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

        let len_sig = rdr.read_u8()?;
        let mut sigs: Vec<Signature> = Vec::with_capacity(len_sig as usize);
        for i in 0..len_sig {
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
        let _length = payload_data.read_u8()?;

        let payload = match action {
            0x01 => {
                VAABody::UpdateGuardianSet(BodyUpdateGuardianSet::deserialize(&mut payload_data)?)
            }
            0x10 => VAABody::Transfer(BodyTransfer::deserialize(&mut payload_data)?),
            _ => {
                return Err(Error::InvalidVAAAction);
            }
        };

        Ok(payload)
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        match self {
            VAABody::Transfer(b) => b.serialize(),
            VAABody::UpdateGuardianSet(b) => b.serialize(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    pub new_index: u32,
    pub new_keys: Vec<[u8; 20]>,
}

#[derive(Clone, Debug, PartialEq)]
pub struct BodyTransfer {
    pub nonce: u32,
    pub source_chain: u8,
    pub target_chain: u8,
    pub source_address: ForeignAddress,
    pub target_address: ForeignAddress,
    pub asset: AssetMeta,
    pub amount: U256,
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

impl BodyTransfer {
    fn deserialize(data: &mut Cursor<&Vec<u8>>) -> Result<BodyTransfer, Error> {
        let nonce = data.read_u32::<BigEndian>()?;
        let source_chain = data.read_u8()?;
        let target_chain = data.read_u8()?;
        let mut source_address: ForeignAddress = ForeignAddress::default();
        data.read_exact(&mut source_address)?;
        let mut target_address: ForeignAddress = ForeignAddress::default();
        data.read_exact(&mut target_address)?;
        let token_chain = data.read_u8()?;
        let mut token_address: ForeignAddress = ForeignAddress::default();
        data.read_exact(&mut token_address)?;

        let mut am_data: [u8; 32] = [0; 32];
        data.read_exact(&mut am_data)?;
        let amount = U256::from_big_endian(&am_data);

        Ok(BodyTransfer {
            nonce,
            source_chain,
            target_chain,
            source_address,
            target_address,
            asset: AssetMeta {
                address: token_address,
                chain: token_chain,
            },
            amount,
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.nonce)?;
        v.write_u8(self.source_chain)?;
        v.write_u8(self.target_chain)?;
        v.write(&self.source_address)?;
        v.write(&self.target_address)?;
        v.write_u8(self.asset.chain)?;
        v.write(&self.asset.address)?;

        let mut am_data: [u8; 32] = [0; 32];
        self.amount.to_big_endian(&mut am_data);
        v.write(&am_data[..])?;

        Ok(v.into_inner())
    }
}

#[cfg(test)]
mod tests {
    use hex;
    use primitive_types::U256;

    use crate::state::AssetMeta;
    use crate::vaa::{BodyTransfer, BodyUpdateGuardianSet, Signature, VAABody, VAA};

    #[test]
    fn serialize_deserialize_vaa_transfer() {
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
            payload: Some(VAABody::Transfer(BodyTransfer {
                nonce: 28,
                source_chain: 1,
                target_chain: 2,
                source_address: [1; 32],
                target_address: [1; 32],
                asset: AssetMeta {
                    address: [2; 32],
                    chain: 8,
                },
                amount: U256::from(3),
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
        let data = hex::decode("010000000001003382c71a4c79e1518a6ce29c91569f6427a60a95696a3515b8c2340b6acffd723315bd1011aa779f22573882a4edfe1b8206548e134871a23f8ba0c1c7d0b5ed0100000bb801190000000101befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }

    #[test]
    fn parse_given_transfer() {
        let vaa = VAA {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![Signature {
                index: 0,
                r: [
                    146, 115, 122, 21, 4, 243, 179, 223, 140, 147, 203, 133, 198, 74, 72, 96, 187,
                    39, 14, 38, 2, 107, 110, 55, 240, 149, 53, 106, 64, 111, 106, 244,
                ],
                s: [
                    57, 198, 178, 233, 119, 95, 161, 198, 102, 149, 37, 240, 110, 218, 176, 51,
                    186, 93, 68, 115, 8, 244, 227, 189, 179, 60, 15, 54, 29, 195, 46, 195,
                ],
                v: 1,
            }],
            timestamp: 1597440008,
            payload: Some(VAABody::Transfer(BodyTransfer {
                nonce: 53,
                source_chain: 1,
                target_chain: 2,
                source_address: [
                    2, 1, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                    0, 0, 0, 0, 0, 0,
                ],
                target_address: [
                    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 144, 248, 191, 106, 71, 159, 50, 14, 173,
                    7, 68, 17, 164, 176, 231, 148, 78, 168, 201, 193,
                ],
                asset: AssetMeta {
                    address: [
                        0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 71, 239, 52, 104, 123, 220, 159, 24,
                        158, 135, 169, 32, 6, 88, 217, 196, 14, 153, 136,
                    ],
                    chain: 1,
                },
                amount: U256::from_dec_str("5000000000000000000").unwrap(),
            })),
        };
        let data = hex::decode("0100000000010092737a1504f3b3df8c93cb85c64a4860bb270e26026b6e37f095356a406f6af439c6b2e9775fa1c6669525f06edab033ba5d447308f4e3bdb33c0f361dc32ec3015f3700081087000000350102020104000000000000000000000000000000000000000000000000000000000000000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1010000000000000000000000000347ef34687bdc9f189e87a9200658d9c40e99880000000000000000000000000000000000000000000000004563918244f40000").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }
}
