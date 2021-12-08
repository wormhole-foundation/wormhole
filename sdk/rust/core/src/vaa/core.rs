//! This module exposes parsers for core bridge VAAs. The main job of the bridge is to forward
//! VAA's to other chains, however governance actions are themselves VAAs and as such the bridge
//! requires parsing Bridge specific VAAs.
//!
//! The core bridge does not define any general VAA's, thus all the payloads in this file are
//! expected to require governance to be executed.

use nom::multi::{
    count,
    fill,
};
use nom::number::complete::{
    u32,
    u8,
};
use nom::number::Endianness;
use nom::IResult;
use primitive_types::U256;

use crate::vaa::{
    parse_fixed,
    GovernanceAction,
};

pub struct GovernanceContractUpgrade {
    pub new_contract: [u8; 32],
}

impl GovernanceAction for GovernanceContractUpgrade {
    const MODULE: &'static [u8] = b"Core";
    const ACTION: u8 = 1;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_contract) = parse_fixed(input)?;
        Ok((i, Self { new_contract }))
    }
}

pub struct GovernanceGuardianSetChange {
    pub new_guardian_set_index: u32,
    pub new_guardian_set:       Vec<[u8; 20]>,
}

impl GovernanceAction for GovernanceGuardianSetChange {
    const MODULE: &'static [u8] = b"Core";
    const ACTION: u8 = 2;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_guardian_set_index) = u32(Endianness::Big)(input)?;
        let (i, guardian_count) = u8(i)?;
        let (i, new_guardian_set) = count(parse_fixed, guardian_count.into())(i)?;
        Ok((
            i,
            Self {
                new_guardian_set_index,
                new_guardian_set,
            },
        ))
    }
}

pub struct GovernanceSetMessageFee {
    pub fee: U256,
}

impl GovernanceAction for GovernanceSetMessageFee {
    const MODULE: &'static [u8] = b"Core";
    const ACTION: u8 = 3;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let mut fee = [0u8; 32];
        let (i, _) = fill(u8, &mut fee)(input)?;
        Ok((
            i,
            Self {
                fee: U256::from_big_endian(&fee),
            },
        ))
    }
}

pub struct GovernanceTransferFees {
    pub amount: U256,
    pub to:     [u8; 32],
}

impl GovernanceAction for GovernanceTransferFees {
    const MODULE: &'static [u8] = b"Core";
    const ACTION: u8 = 4;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let mut amount = [0u8; 32];
        let (i, _) = fill(u8, &mut amount)(input)?;
        let (i, to) = parse_fixed(i)?;
        Ok((
            i,
            Self {
                amount: U256::from_big_endian(&amount),
                to,
            },
        ))
    }
}
