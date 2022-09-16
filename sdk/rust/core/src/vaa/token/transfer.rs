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
    primitive_types::U256,
};


/// Transfer is a message containing specifics detailing a token lock up on a sending chain. Chains
/// that are attempting to initiate a transfer must lock up tokens in some manner, such as in a
/// custody account or via burning, before emitting this message.
#[derive(PartialEq, Eq, Debug, Clone)]
pub struct Transfer {
    /// Amount being transferred (big-endian uint256)
    pub amount: U256,

    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: [u8; 32],

    /// Chain ID of the token
    pub token_chain: Chain,

    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: [u8; 32],

    /// Chain ID of the recipient
    pub to_chain: Chain,

    /// Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
    pub fee: U256,
}

impl Transfer {
    #[inline]
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        // Parse Transfer Payload.
        let (i, (amount, token_address, token_chain, to, to_chain, fee)) = nom::sequence::tuple((
            parse_fixed::<32>,
            parse_fixed,
            parse_chain,
            parse_fixed,
            parse_chain,
            parse_fixed::<32>,
        ))(input)?;

        Ok((
            i,
            Action::Transfer(Self {
                amount: U256::from_big_endian(&amount),
                token_address,
                token_chain,
                to,
                to_chain,
                fee: U256::from_big_endian(&fee),
            }),
        ))
    }
}
