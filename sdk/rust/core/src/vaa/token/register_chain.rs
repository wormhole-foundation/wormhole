use {
    super::Action,
    crate::{
        vaa::{
            parse_chain,
            parse_fixed,
        },
        Chain,
    },
    nom::IResult,
};


#[derive(PartialEq, Eq, Debug)]
pub struct RegisterChain {
    pub chain:            Chain,
    pub emitter:          Chain,
    pub endpoint_address: [u8; 32],
}

impl RegisterChain {
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        let (i, (chain, emitter, endpoint_address)) =
            nom::sequence::tuple((parse_chain, parse_chain, parse_fixed))(input)?;

        Ok((
            i,
            Action::RegisterChain(Self {
                chain,
                emitter,
                endpoint_address,
            }),
        ))
    }
}
