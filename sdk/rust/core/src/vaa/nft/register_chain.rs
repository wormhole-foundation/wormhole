//! Parser for the RegisterChain action for the NFT contract.

use {
    super::Action,
    crate::{
        parse_chain,
        vaa::parse_fixed,
        Chain,
        GovHeader,
    },
    nom::IResult,
};


#[derive(PartialEq, Eq, Debug)]
pub struct RegisterChain {
    pub header:           GovHeader,
    pub emitter:          Chain,
    pub endpoint_address: [u8; 32],
}

impl RegisterChain {
    #[inline]
    pub fn parse(i: &[u8]) -> IResult<&[u8], Action> {
        let (i, header) = GovHeader::parse(i)?;
        let (i, emitter) = parse_chain(i)?;
        let (i, endpoint_address) = parse_fixed(i)?;
        Ok((
            i,
            Action::RegisterChain(Self {
                header,
                emitter,
                endpoint_address,
            }),
        ))
    }
}
