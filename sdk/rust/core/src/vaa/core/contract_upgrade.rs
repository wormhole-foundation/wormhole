//! Parser for the ContractUpgrade action for the core contract.

use {
    super::Action,
    crate::{
        vaa::parse_fixed,
        GovHeader,
    },
    nom::IResult,
};


#[derive(Debug, PartialEq, Eq)]
pub struct ContractUpgrade {
    pub header:       GovHeader,
    pub new_contract: [u8; 32],
}

impl ContractUpgrade {
    #[inline]
    pub fn parse(i: &[u8], header: GovHeader) -> IResult<&[u8], Action> {
        let (i, new_contract) = parse_fixed(i)?;
        Ok((
            i,
            Action::ContractUpgrade(Self {
                header,
                new_contract,
            }),
        ))
    }
}


#[cfg(test)]
mod testing {
    use {
        super::*,
        crate::{
            Chain,
            VAA,
        },
        byteorder::{
            BigEndian,
            ReadBytesExt,
        },
        std::io::{
            Cursor,
            Read,
        },
    };

    // Original ContractUpgrade Parsing Code. Used to compare current code to old for parity.
    pub fn legacy_deserialize(data: &[u8]) -> std::result::Result<ContractUpgrade, std::io::Error> {
        let mut c = Cursor::new(data);
        let mut module = [0u8; 32];
        c.read_exact(&mut module)?;
        let action = c.read_u8()?;
        let target = c.read_u16::<BigEndian>()?;
        let mut addr = [0u8; 32];
        c.read_exact(&mut addr)?;
        Ok(ContractUpgrade {
            header:       GovHeader {
                module,
                action,
                target: target.into(),
            },
            new_contract: addr,
        })
    }

    const TEST_VAA: [u8; 190] = hex_literal::hex!("01000000000100e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe7009000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004c5d05a0000000000000000000000000000000000000000000000000000000000436f726501000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2");

    // Check `ContractUpgrade::parse` against the legacy implementation.
    #[test]
    fn check_parse_parity() {
        let vaa = VAA::from_bytes(TEST_VAA).unwrap();
        let new = <Action as crate::Action>::from_vaa(&vaa, Chain::Fantom).unwrap();
        let old = legacy_deserialize(&vaa.payload).unwrap();
        assert_eq!(new, Action::ContractUpgrade(old));
    }

    #[bench]
    fn bench_parse(b: &mut test::Bencher) {
        let vaa = VAA::from_bytes(TEST_VAA).unwrap();
        b.iter(|| {
            let _ = <Action as crate::Action>::from_vaa(&vaa, Chain::Fantom).unwrap();
        });
    }

    #[bench]
    fn bench_old_parse(b: &mut test::Bencher) {
        let vaa = VAA::from_bytes(TEST_VAA).unwrap();
        b.iter(|| {
            let _ = legacy_deserialize(&vaa.payload).unwrap();
        });
    }
}
