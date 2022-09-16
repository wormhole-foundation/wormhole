//! Parser for the SetMessageFee action for the core contract.

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
pub struct SetMessageFee {
    pub header: GovHeader,
    pub fee:    U256,
}

impl SetMessageFee {
    #[inline]
    pub fn parse(i: &[u8], header: GovHeader) -> IResult<&[u8], Action> {
        let (i, fee): (_, [u8; 32]) = parse_fixed(i)?;
        Ok((
            i,
            Action::SetMessageFee(Self {
                header,
                fee: U256::from_big_endian(&fee),
            }),
        ))
    }
}
