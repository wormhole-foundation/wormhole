use {
    super::Action,
    crate::{
        vaa::{
            parse_chain,
            parse_fixed,
        },
        Chain,
    },
    nom::{
        combinator::rest,
        IResult,
    },
    primitive_types::U256,
};


/// TransferWithPayload is similar to Transfer but also carries an arbitrary message payload.
#[derive(PartialEq, Eq, Debug, Clone)]
pub struct TransferWithPayload {
    /// Amount being transferred (big-endian uint256)
    pub amount:        U256,
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: [u8; 32],
    /// Chain ID of the token
    pub token_chain:   Chain,
    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to:            [u8; 32],
    /// Chain ID of the recipient
    pub to_chain:      Chain,
    /// From address acts as the identity submitting the payload.
    pub from_address:  [u8; 32],
    /// Arbitrary Payload to send to the target chain.
    pub payload:       Vec<u8>,
}

impl TransferWithPayload {
    #[inline]
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        // Parse Transfer Payload.
        let (i, (amount, token_address, token_chain, to, to_chain, from_address, payload)) =
            nom::sequence::tuple((
                parse_fixed::<32>,
                parse_fixed,
                parse_chain,
                parse_fixed,
                parse_chain,
                parse_fixed,
                rest,
            ))(input)?;

        Ok((
            i,
            Action::TransferWithPayload(Self {
                amount: U256::from_big_endian(&amount),
                payload: payload.to_vec(),
                token_address,
                token_chain,
                to,
                to_chain,
                from_address,
            }),
        ))
    }
}
