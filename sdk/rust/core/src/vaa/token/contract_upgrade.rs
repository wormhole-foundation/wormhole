use {
    super::Action,
    crate::vaa::parse_fixed,
    nom::IResult,
};


#[derive(PartialEq, Eq, Debug)]
pub struct ContractUpgrade {
    pub new_contract: [u8; 32],
}

impl ContractUpgrade {
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        let (i, new_contract) = parse_fixed(input)?;
        Ok((i, Action::ContractUpgrade(Self { new_contract })))
    }
}
