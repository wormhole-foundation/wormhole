use std::io::{Cursor, Read, Write};

use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use primitive_types::U256;
use sha3::Digest;

use crate::error::Error;
use crate::error::Error::InvalidVAAFormat;
use crate::state::AssetMeta;
use crate::syscalls::{sol_verify_schnorr, RawKey, SchnorrifyInput};

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

        let schnorr_input =
            SchnorrifyInput::new(*guardian_key, hash, self.signature_sig, self.signature_addr);
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

#[derive(Copy, Clone, Debug, PartialEq)]
pub struct BodyUpdateGuardianSet {
    pub new_index: u32,
    pub new_key: RawKey,
}

#[derive(Copy, Clone, Debug, PartialEq)]
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
        let mut new_key_x: [u8; 32] = [0; 32];
        data.read(&mut new_key_x)?;
        let new_key_y_parity = match data.read_u8()? {
            0 => false,
            1 => true,
            _ => return Err(InvalidVAAFormat),
        };

        let new_index = data.read_u32::<BigEndian>()?;

        Ok(BodyUpdateGuardianSet {
            new_index,
            new_key: RawKey {
                x: new_key_x,
                y_parity: new_key_y_parity,
            },
        })
    }

    fn serialize(&self) -> Result<Vec<u8>, Error> {
        let mut v: Cursor<Vec<u8>> = Cursor::new(Vec::new());
        v.write(&self.new_key.x)?;
        v.write_u8({
            match self.new_key.y_parity {
                false => 0,
                true => 1,
            }
        })?;
        v.write_u32::<BigEndian>(self.new_index)?;

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
    use crate::syscalls::RawKey;
    use crate::vaa::{BodyTransfer, BodyUpdateGuardianSet, VAABody, VAA};

    #[test]
    fn serialize_deserialize_vaa_transfer() {
        let vaa = VAA {
            version: 8,
            guardian_set_index: 3,
            signature_sig: [7; 32],
            signature_addr: [9; 20],
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
            signature_sig: [7; 32],
            signature_addr: [9; 20],
            timestamp: 83,
            payload: Some(VAABody::UpdateGuardianSet(BodyUpdateGuardianSet {
                new_index: 29,
                new_key: RawKey {
                    x: [2; 32],
                    y_parity: true,
                },
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
            guardian_set_index: 9,
            signature_sig: [
                2, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0,
            ],
            signature_addr: [1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
            timestamp: 2837,
            payload: Some(VAABody::UpdateGuardianSet(BodyUpdateGuardianSet {
                new_index: 2,
                new_key: RawKey {
                    x: [
                        34, 23, 130, 103, 189, 101, 144, 104, 196, 19, 115, 119, 37, 80, 123, 46,
                        218, 191, 167, 75, 3, 40, 130, 168, 218, 203, 128, 99, 120, 238, 102, 1,
                    ],
                    y_parity: true,
                },
            })),
        };
        let data = hex::decode("01000000090208000000000000000000000000000000000000000000000000000000000000010203040000000000000000000000000000000000000b15012522178267bd659068c413737725507b2edabfa74b032882a8dacb806378ee66010100000002").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }

    #[test]
    fn parse_given_transfer() {
        let vaa = VAA {
            version: 1,
            guardian_set_index: 9,
            signature_sig: [
                2, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0,
            ],
            signature_addr: [1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
            timestamp: 2837,
            payload: Some(VAABody::Transfer(BodyTransfer {
                nonce: 38,
                source_chain: 2,
                target_chain: 1,
                source_address: [
                    2, 1, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                    0, 0, 0, 0, 0, 0,
                ],
                target_address: [
                    2, 1, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                    0, 0, 0, 0, 0, 0,
                ],
                asset: AssetMeta {
                    address: [
                        9, 2, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                        0, 0, 0, 0, 0, 0, 0,
                    ],
                    chain: 9,
                },
                amount: U256::from(29),
            })),
        };
        let data = hex::decode("01000000090208000000000000000000000000000000000000000000000000000000000000010203040000000000000000000000000000000000000b15108700000026020102010400000000000000000000000000000000000000000000000000000000000201030000000000000000000000000000000000000000000000000000000000090902040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001d").unwrap();
        let parsed_vaa = VAA::deserialize(data.as_slice()).unwrap();
        assert_eq!(vaa, parsed_vaa);

        let rec_data = parsed_vaa.serialize().unwrap();
        assert_eq!(data, rec_data);
    }
}
