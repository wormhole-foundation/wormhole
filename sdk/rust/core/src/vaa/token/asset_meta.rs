//! Parsers for the Transfer action in the Token contract.

use {
    super::Action,
    crate::{
        parse_fixed_utf8,
        vaa::{
            parse_chain,
            parse_fixed,
            ShortUTFString,
        },
        Chain,
    },
    nom::{
        number::complete::u8,
        IResult,
    },
};


/// AssetMeta contains information about a token and its origin chain.
#[derive(PartialEq, Eq, Debug)]
pub struct AssetMeta {
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: [u8; 32],
    /// Chain ID of the token
    pub token_chain:   Chain,
    /// Number of decimals in the token.
    pub decimals:      u8,
    /// Symbol of the token.
    pub symbol:        ShortUTFString,
    /// Name of the token.
    pub name:          ShortUTFString,
}

impl AssetMeta {
    #[inline]
    pub fn parse(input: &[u8]) -> IResult<&[u8], Action> {
        // Parse Payload.
        let (i, (token_address, token_chain, decimals, symbol, name)) = nom::sequence::tuple((
            parse_fixed,
            parse_chain,
            u8,
            parse_fixed::<32>,
            parse_fixed::<32>,
        ))(input)?;

        // Name & Symbol should be UTF-8 strings, attempt to parse them by removing invalid bytes.
        let name = parse_fixed_utf8::<_, 32>(name).unwrap();
        let symbol = parse_fixed_utf8::<_, 32>(symbol).unwrap();

        Ok((
            i,
            Action::AssetMeta(Self {
                token_address,
                token_chain,
                decimals,
                symbol,
                name,
            }),
        ))
    }
}
