//! This module exposes parsers for core bridge VAAs. The main job of the bridge is to forward
//! VAA's to other chains, however governance actions are themselves VAAs and as such the bridge
//! requires parsing Bridge specific VAAs.
//!
//! The core bridge does not define any general VAA's, thus all the payloads in this file are
//! expected to require governance to be executed.

use {
    crate::vaa::{
        parse_fixed,
        Action,
        GovHeader,
    },
    nom::{
        error::ErrorKind,
        multi::{
            count,
            fill,
        },
        number::{
            complete::{
                u32,
                u8,
            },
            Endianness,
        },
        IResult,
    },
    primitive_types::U256,
};

pub enum CoreAction {
    ContractUpgrade(ContractUpgrade),
    GuardianSetChange(GuardianSetChange),
    SetMessageFee(SetMessageFee),
    TransferFees(TransferFees),
}

impl Action for CoreAction {
    // Module: 000..Core
    const MODULE: [u8; 32] =
        hex_literal::hex!("00000000000000000000000000000000000000000000000000000000436f7265");

    #[inline]
    fn parse<'a, 'b>(i: &'a [u8], header: &'b GovHeader) -> IResult<&'a [u8], Self> {
        match header.action {
            1 => ContractUpgrade::parse(i).map(|(i, r)| (i, Self::ContractUpgrade(r))),
            2 => GuardianSetChange::parse(i).map(|(i, r)| (i, Self::GuardianSetChange(r))),
            3 => SetMessageFee::parse(i).map(|(i, r)| (i, Self::SetMessageFee(r))),
            4 => TransferFees::parse(i).map(|(i, r)| (i, Self::TransferFees(r))),
            _ => Err(nom::Err::Error(nom::error_position!(i, ErrorKind::NoneOf))),
        }
    }
}

pub struct ContractUpgrade {
    pub new_contract: [u8; 32],
}

impl ContractUpgrade {
    #[inline]
    fn parse(i: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_contract) = parse_fixed(i)?;
        Ok((i, Self { new_contract }))
    }
}

pub struct GuardianSetChange {
    pub new_guardian_set_index: u32,
    pub new_guardian_set:       Vec<[u8; 20]>,
}

impl GuardianSetChange {
    #[inline]
    fn parse(i: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_guardian_set_index) = u32(Endianness::Big)(i)?;
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

pub struct SetMessageFee {
    pub fee: U256,
}

impl SetMessageFee {
    #[inline]
    fn parse(i: &[u8]) -> IResult<&[u8], Self> {
        let mut fee = [0u8; 32];
        let (i, _) = fill(u8, &mut fee)(i)?;
        Ok((
            i,
            Self {
                fee: U256::from_big_endian(&fee),
            },
        ))
    }
}

pub struct TransferFees {
    pub amount: U256,
    pub to:     [u8; 32],
}

impl TransferFees {
    #[inline]
    fn parse(i: &[u8]) -> IResult<&[u8], Self> {
        let mut amount = [0u8; 32];
        let (i, _) = fill(u8, &mut amount)(i)?;
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
