//! Parser for the Transfer action in the NFT contraccontract
use {
    super::Action,
    crate::{
        parse_chain,
        parse_fixed_utf8,
        vaa::{
            parse_fixed,
            ShortUTFString,
        },
        Chain,
    },
    nom::{
        bytes::complete::take,
        combinator::flat_map,
        number::complete::u8,
        IResult,
    },
    primitive_types::U256,
};


/// The Transfer message contains specifics detailing a token lock up on a sending chain. Chains
/// that are attempting to initiate a transfer must lock up tokens in some manner, such as in a
/// custody account or via burning, before emitting this message.
#[derive(Debug, PartialEq, Eq)]
pub struct Transfer {
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub nft_address: [u8; 32],

    /// Chain ID of the token
    pub nft_chain: Chain,

    /// Symbol of the token
    pub symbol: ShortUTFString,

    /// Name of the token
    pub name: ShortUTFString,

    /// TokenID of the token (big-endian uint256)
    pub token_id: U256,

    /// URI of the token metadata
    pub uri: ShortUTFString,

    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: [u8; 32],

    /// Chain ID of the recipient
    pub to_chain: Chain,
}

impl Transfer {
    #[inline]
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        let (i, (nft_address, nft_chain, symbol, name, token_id, uri, to, to_chain)) =
            nom::sequence::tuple((
                parse_fixed,
                parse_chain,
                parse_fixed::<32>,
                parse_fixed::<32>,
                parse_fixed::<32>,
                flat_map(u8, take),
                parse_fixed,
                parse_chain,
            ))(input)?;

        let symbol = parse_fixed_utf8::<_, 32>(symbol).unwrap();
        let name = parse_fixed_utf8::<_, 32>(name).unwrap();

        Ok((
            i,
            Action::Transfer(Transfer {
                nft_address,
                nft_chain,
                symbol,
                name,
                token_id: U256::from_big_endian(&token_id),
                uri: std::str::from_utf8(uri).unwrap().to_string(),
                to,
                to_chain,
            }),
        ))
    }
}
