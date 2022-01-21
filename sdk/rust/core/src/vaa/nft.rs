//! This module exposes parsers for NFT bridge VAAs. Token bridging relies on VAA's that indicate
//! custody/lockup/burn events in order to maintain token parity between multiple chains. These
//! parsers can be used to read these VAAs. It also defines the Governance actions that this module
//! supports, namely contract upgrades and chain registrations.

use nom::bytes::complete::take;
use nom::combinator::verify;
use nom::number::complete::u8;
use nom::{
    Finish,
    IResult,
};
use primitive_types::U256;
use std::str::from_utf8;

use crate::vaa::{
    parse_chain,
    parse_fixed,
    GovernanceAction,
};
use crate::vaa::ShortUTFString;
use crate::{
    Chain,
    parse_fixed_utf8,
    WormholeError,
};

/// Transfer is a message containing specifics detailing a token lock up on a sending chain. Chains
/// that are attempting to initiate a transfer must lock up tokens in some manner, such as in a
/// custody account or via burning, before emitting this message.
#[derive(PartialEq, Debug, Clone)]
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
    pub fn from_bytes<T: AsRef<[u8]>>(input: T) -> Result<Self, WormholeError> {
        match parse_payload_transfer(input.as_ref()).finish() {
            Ok(input) => Ok(input.1),
            Err(e) => Err(WormholeError::ParseError(e.code as usize)),
        }
    }
}

fn parse_payload_transfer(input: &[u8]) -> IResult<&[u8], Transfer> {
    // Parse Payload
    let (i, _) = verify(u8, |&s| s == 0x1)(input.as_ref())?;
    let (i, nft_address) = parse_fixed(i)?;
    let (i, nft_chain) = parse_chain(i)?;
    let (i, symbol): (_, [u8; 32]) = parse_fixed(i)?;
    let (i, name): (_, [u8; 32]) = parse_fixed(i)?;
    let (i, token_id): (_, [u8; 32]) = parse_fixed(i)?;
    let (i, uri_len) = u8(i)?;
    let (i, uri) = take(uri_len)(i)?;
    let (i, to) = parse_fixed(i)?;
    let (i, to_chain) = parse_chain(i)?;

    // Name/Symbol and URI should be UTF-8 strings, attempt to parse the first two by removing
    // invalid bytes -- for the latter, assume UTF-8 and fail if unparseable.
    let name = parse_fixed_utf8::<_, 32>(name).unwrap();
    let symbol = parse_fixed_utf8::<_, 32>(symbol).unwrap();
    let uri = from_utf8(uri).unwrap().to_string();

    Ok((
        i,
        Transfer {
            nft_address,
            nft_chain,
            symbol,
            name,
            token_id: U256::from_big_endian(&token_id),
            uri,
            to,
            to_chain,
        },
    ))
}

#[derive(PartialEq, Debug)]
pub struct GovernanceRegisterChain {
    pub emitter:          Chain,
    pub endpoint_address: [u8; 32],
}

impl GovernanceAction for GovernanceRegisterChain {
    const MODULE: &'static [u8] = b"NFTBridge";
    const ACTION: u8 = 1;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, emitter) = parse_chain(input)?;
        let (i, endpoint_address) = parse_fixed(i)?;
        Ok((
            i,
            Self {
                emitter,
                endpoint_address,
            },
        ))
    }
}

#[derive(PartialEq, Debug)]
pub struct GovernanceContractUpgrade {
    pub new_contract: [u8; 32],
}

impl GovernanceAction for GovernanceContractUpgrade {
    const MODULE: &'static [u8] = b"NFTBridge";
    const ACTION: u8 = 2;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_contract) = parse_fixed(input)?;
        Ok((i, Self { new_contract }))
    }
}
