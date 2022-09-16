//! Parser for the TransferFees for the core contract.

use {
    super::Action,
    crate::{
        parse_fixed,
        GovHeader,
    },
    nom::IResult,
    primitive_types::U256,
};


#[derive(Debug, PartialEq, Eq)]
pub struct TransferFees {
    pub header: GovHeader,
    pub amount: U256,
    pub to:     [u8; 32],
}

impl TransferFees {
    #[inline]
    pub fn parse(i: &[u8], header: GovHeader) -> IResult<&[u8], Action> {
        let (i, amount): (_, [u8; 32]) = parse_fixed(i)?;
        let (i, to) = parse_fixed(i)?;
        Ok((
            i,
            Action::TransferFees(Self {
                header,
                amount: U256::from_big_endian(&amount),
                to,
            }),
        ))
    }
}
