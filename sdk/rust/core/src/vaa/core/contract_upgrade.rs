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
