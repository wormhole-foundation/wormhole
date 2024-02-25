use byteorder::{BigEndian, ReadBytesExt};
use std::io::{Cursor, Read};

// akin to https://github.com/wormhole-foundation/example-native-token-transfers/blob/main/evm/src/libraries/EndpointStructs.sol
// should probably be covered in the `ntt-messages` crate

pub enum ManagerMode {
    LOCKING = 0,
    BURNING = 1,
}

pub struct EndpointTransfer {}

impl EndpointTransfer {
    pub const PREFIX: [u8; 4] = [0x99, 0x45, 0xFF, 0x10]; // 0x99'E''W''H'
}

pub struct EndpointInit {
    pub manager_address: [u8; 32],
    pub manager_mode: u8,
    pub token_address: [u8; 32],
    pub token_decimals: u8,
}

impl EndpointInit {
    pub const PREFIX: [u8; 4] = [0xc8, 0x3e, 0x3d, 0x2e]; // bytes4(keccak256("WormholeEndpointInit"))

    pub fn deserialize(data: &[u8]) -> std::result::Result<EndpointInit, std::io::Error> {
        let mut rdr = Cursor::new(data);
        Self::deserialize_from_reader(&mut rdr)
    }

    pub fn deserialize_from_reader(
        rdr: &mut Cursor<&[u8]>,
    ) -> std::result::Result<EndpointInit, std::io::Error> {
        let mut endpoint_identifier = [0u8; 4];
        rdr.read_exact(&mut endpoint_identifier)?;
        if endpoint_identifier != Self::PREFIX {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "PayloadMismatch",
            ));
        }

        let mut manager_address = [0u8; 32];
        rdr.read_exact(&mut manager_address)?;

        let manager_mode = rdr.read_u8()?;

        let mut token_address = [0u8; 32];
        rdr.read_exact(&mut token_address)?;

        let token_decimals = rdr.read_u8()?;

        if rdr.position() != rdr.get_ref().len() as u64 {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "InvalidPayloadLength",
            ));
        }

        Ok(EndpointInit {
            manager_address,
            manager_mode,
            token_address,
            token_decimals,
        })
    }
}

pub struct EndpointRegister {
    pub endpoint_chain_id: u16,
    pub endpoint_address: [u8; 32],
}

impl EndpointRegister {
    pub const PREFIX: [u8; 4] = [0xd0, 0xd2, 0x92, 0xf1]; // bytes4(keccak256("WormholeSiblingRegistration"))

    pub fn deserialize(data: &[u8]) -> std::result::Result<EndpointRegister, std::io::Error> {
        let mut rdr = Cursor::new(data);
        Self::deserialize_from_reader(&mut rdr)
    }

    pub fn deserialize_from_reader(
        rdr: &mut Cursor<&[u8]>,
    ) -> std::result::Result<EndpointRegister, std::io::Error> {
        let mut endpoint_identifier = [0u8; 4];
        rdr.read_exact(&mut endpoint_identifier)?;
        if endpoint_identifier != Self::PREFIX {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "PayloadMismatch",
            ));
        }

        let endpoint_chain_id = rdr.read_u16::<BigEndian>()?;

        let mut endpoint_address = [0u8; 32];
        rdr.read_exact(&mut endpoint_address)?;

        if rdr.position() != rdr.get_ref().len() as u64 {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "InvalidPayloadLength",
            ));
        }

        Ok(EndpointRegister {
            endpoint_chain_id,
            endpoint_address,
        })
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    pub fn lock_init() {
        // c83e3d2e000000000000000000000000bb807f76cda53b1b4256e1b6f33bb46be36508e3000000000000000000000000002a68f967bfa230780a385175d0c86ae4048d3096120000000000000000000000000000000000000000000000000000
        let payload = [
            0xc8, 0x3e, 0x3d, 0x2e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0xbb, 0x80, 0x7f, 0x76, 0xcd, 0xa5, 0x3b, 0x1b, 0x42, 0x56, 0xe1, 0xb6,
            0xf3, 0x3b, 0xb4, 0x6b, 0xe3, 0x65, 0x08, 0xe3, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2a, 0x68, 0xf9, 0x67, 0xbf, 0xa2, 0x30,
            0x78, 0x0a, 0x38, 0x51, 0x75, 0xd0, 0xc8, 0x6a, 0xe4, 0x04, 0x8d, 0x30, 0x96, 0x12,
        ];
        let init = EndpointInit::deserialize(&payload).unwrap();

        assert_eq!(
            init.manager_address,
            [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xbb, 0x80,
                0x7f, 0x76, 0xcd, 0xa5, 0x3b, 0x1b, 0x42, 0x56, 0xe1, 0xb6, 0xf3, 0x3b, 0xb4, 0x6b,
                0xe3, 0x65, 0x08, 0xe3
            ]
        );
        assert_eq!(init.manager_mode, ManagerMode::LOCKING as u8);
        assert_eq!(
            init.token_address,
            [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2a, 0x68,
                0xf9, 0x67, 0xbf, 0xa2, 0x30, 0x78, 0x0a, 0x38, 0x51, 0x75, 0xd0, 0xc8, 0x6a, 0xe4,
                0x04, 0x8d, 0x30, 0x96
            ]
        );
        assert_eq!(init.token_decimals, 18);
    }

    #[test]
    pub fn burn_init() {
        // c83e3d2e0000000000000000000000001fc14f21b27579f4f23578731cd361cca8aa39f701000000000000000000000000eb502b1d35e975321b21cce0e8890d20a7eb289d120000000000000000000000000000000000000000000000000000
        let payload = [
            0xc8, 0x3e, 0x3d, 0x2e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x1f, 0xc1, 0x4f, 0x21, 0xb2, 0x75, 0x79, 0xf4, 0xf2, 0x35, 0x78, 0x73,
            0x1c, 0xd3, 0x61, 0xcc, 0xa8, 0xaa, 0x39, 0xf7, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xeb, 0x50, 0x2b, 0x1d, 0x35, 0xe9, 0x75,
            0x32, 0x1b, 0x21, 0xcc, 0xe0, 0xe8, 0x89, 0x0d, 0x20, 0xa7, 0xeb, 0x28, 0x9d, 0x12,
        ];
        let init = EndpointInit::deserialize(&payload).unwrap();

        assert_eq!(
            init.manager_address,
            [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1f, 0xc1,
                0x4f, 0x21, 0xb2, 0x75, 0x79, 0xf4, 0xf2, 0x35, 0x78, 0x73, 0x1c, 0xd3, 0x61, 0xcc,
                0xa8, 0xaa, 0x39, 0xf7,
            ]
        );
        assert_eq!(init.manager_mode, ManagerMode::BURNING as u8);
        assert_eq!(
            init.token_address,
            [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xeb, 0x50,
                0x2b, 0x1d, 0x35, 0xe9, 0x75, 0x32, 0x1b, 0x21, 0xcc, 0xe0, 0xe8, 0x89, 0x0d, 0x20,
                0xa7, 0xeb, 0x28, 0x9d,
            ]
        );
        assert_eq!(init.token_decimals, 18);
    }

    #[test]
    pub fn register() {
        let payload = [
            0xd0, 0xd2, 0x92, 0xf1, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x1f, 0xc1, 0x4f, 0x21, 0xb2, 0x75, 0x79, 0xf4, 0xf2, 0x35,
            0x78, 0x73, 0x1c, 0xd3, 0x61, 0xcc, 0xa8, 0xaa, 0x39, 0xf7,
        ];
        let register = EndpointRegister::deserialize(&payload).unwrap();

        assert_eq!(register.endpoint_chain_id, 1);
        assert_eq!(
            register.endpoint_address,
            [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1f, 0xc1,
                0x4f, 0x21, 0xb2, 0x75, 0x79, 0xf4, 0xf2, 0x35, 0x78, 0x73, 0x1c, 0xd3, 0x61, 0xcc,
                0xa8, 0xaa, 0x39, 0xf7,
            ]
        );
    }
}
