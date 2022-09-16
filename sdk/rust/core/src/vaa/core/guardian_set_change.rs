use {
    super::Action,
    crate::{
        vaa::parse_fixed,
        GovHeader,
    },
    nom::{
        combinator::flat_map,
        multi::count,
        number::{
            complete::{
                u32,
                u8,
            },
            Endianness,
        },
        IResult,
    },
};


#[derive(Debug, PartialEq, Eq)]
pub struct GuardianSetChange {
    pub header:                 GovHeader,
    pub new_guardian_set_index: u32,
    pub new_guardian_set:       Vec<[u8; 20]>,
}

impl GuardianSetChange {
    #[inline]
    pub fn parse(i: &[u8], header: GovHeader) -> IResult<&[u8], Action> {
        let (i, new_guardian_set_index) = u32(Endianness::Big)(i)?;
        let (i, new_guardian_set) = flat_map(u8, |c| count(parse_fixed, c.into()))(i)?;
        Ok((
            i,
            Action::GuardianSetChange(Self {
                header,
                new_guardian_set_index,
                new_guardian_set,
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
    pub fn legacy_deserialize(
        data: &[u8],
    ) -> std::result::Result<GuardianSetChange, std::io::Error> {
        let mut c = Cursor::new(data);
        let mut module = [0u8; 32];
        c.read_exact(&mut module)?;
        let action = c.read_u8()?;
        let target = c.read_u16::<BigEndian>()?;
        let new_index = c.read_u32::<BigEndian>()?;
        let keys_len = c.read_u8()?;
        let mut keys = Vec::with_capacity(keys_len as usize);
        for _ in 0..keys_len {
            let mut key: [u8; 20] = [0; 20];
            c.read_exact(&mut key)?;
            keys.push(key);
        }
        Ok(GuardianSetChange {
            header:                 GovHeader {
                module,
                action,
                target: target.into(),
            },
            new_guardian_set_index: new_index,
            new_guardian_set:       keys,
        })
    }

    const TEST_VAA: [u8; 543] = hex_literal::hex!("
        010000000001007ac31b282c2aeeeb37f3385ee0de5f8e421d30b9e5ae8ba3d4375c1c77a86e77159bb697d9c456d6f8c02d22a94b1279b65b0d6a99
        57e7d3857423845ac758e300610ac1d20000000300010000000000000000000000000000000000000000000000000000000000000004000000000000
        05390000000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570
        746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d
        39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd49
        58a7cdeb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0f
        ea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd473141
        2f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027
        c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99
        c8348d
    ");

    // Check `ContractUpgrade::parse` against the legacy implementation.
    #[test]
    fn check_parse_parity() {
        let vaa = VAA::from_bytes(TEST_VAA).unwrap();
        let new = <Action as crate::Action>::from_vaa(&vaa, Chain::Unknown(63519)).unwrap();
        let old = legacy_deserialize(&vaa.payload).unwrap();
        assert_eq!(new, Action::GuardianSetChange(old));
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
