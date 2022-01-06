//! This module exposes parsers for token bridge VAAs. Token bridging relies on VAA's that indicate
//! custody/lockup/burn events in order to maintain token parity between multiple chains. These
//! parsers can be used to read these VAAs. It also defines the Governance actions that this module
//! supports, namely contract upgrades and chain registrations.

use nom::combinator::verify;
use nom::multi::fill;
use nom::number::complete::u8;
use nom::{
    Finish,
    IResult,
};
use primitive_types::U256;

use crate::vaa::{
    GovernanceAction,
    parse_chain,
    parse_fixed,
    ShortUTFString,
};
use crate::{
    parse_fixed_utf8,
    Chain,
    WormholeError,
};

/// Transfer is a message containing specifics detailing a token lock up on a sending chain. Chains
/// that are attempting to initiate a transfer must lock up tokens in some manner, such as in a
/// custody account or via burning, before emitting this message.
#[derive(PartialEq, Debug, Clone)]
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
    pub fn from_bytes<T: AsRef<[u8]>>(input: T) -> Result<Self, WormholeError> {
        match parse_payload_transfer(input.as_ref()).finish() {
            Ok(input) => Ok(input.1),
            Err(e) => Err(WormholeError::ParseError(e.code as usize)),
        }
    }
}

fn parse_payload_transfer(input: &[u8]) -> IResult<&[u8], Transfer> {
    // Parser Buffers.
    let mut amount = [0u8; 32];
    let mut fee = [0u8; 32];

    // Parse Payload.
    let (i, _) = verify(u8, |&s| s == 0x1)(input)?;
    let (i, _) = fill(u8, &mut amount)(i)?;
    let (i, token_address) = parse_fixed(i)?;
    let (i, token_chain) = parse_chain(i)?;
    let (i, to) = parse_fixed(i)?;
    let (i, to_chain) = parse_chain(i)?;
    let (i, _) = fill(u8, &mut fee)(i)?;

    Ok((
        i,
        Transfer {
            amount: U256::from_big_endian(&amount),
            token_address,
            token_chain,
            to,
            to_chain,
            fee: U256::from_big_endian(&fee),
        },
    ))
}

#[derive(PartialEq, Debug)]
pub struct AssetMeta {
    /// Address of the original token on the source chain.
    pub token_address: [u8; 32],

    /// Source Chain ID.
    pub token_chain: Chain,

    /// Number of decimals the source token has on its origin chain.
    pub decimals: u8,

    /// Ticker Symbol for the token on its origin chain.
    pub symbol: ShortUTFString,

    /// Full Token name for the token on its origin chain.
    pub name: ShortUTFString,
}

impl AssetMeta {
    pub fn from_bytes<T: AsRef<[u8]>>(input: T) -> Result<Self, WormholeError> {
        match parse_payload_asset_meta(input.as_ref()).finish() {
            Ok(input) => Ok(input.1),
            Err(e) => Err(WormholeError::ParseError(e.code as usize)),
        }
    }
}

fn parse_payload_asset_meta(input: &[u8]) -> IResult<&[u8], AssetMeta> {
    // Parse Payload.
    let (i, _) = verify(u8, |&s| s == 0x2)(input.as_ref())?;
    let (i, token_address) = parse_fixed(i)?;
    let (i, token_chain) = parse_chain(i)?;
    let (i, decimals) = u8(i)?;
    let (i, symbol): (_, [u8; 32]) = parse_fixed(i)?;
    let (i, name): (_, [u8; 32]) = parse_fixed(i)?;

    // Name/Symbol should be UTF-8 strings, attempt to parse them by removing invalid bytes.
    let symbol = parse_fixed_utf8::<_, 32>(symbol).unwrap();
    let name = parse_fixed_utf8::<_, 32>(name).unwrap();

    Ok((
        i,
        AssetMeta {
            token_address,
            token_chain,
            decimals,
            symbol,
            name,
        },
    ))
}

#[derive(PartialEq, Debug)]
pub struct GovernanceRegisterChain {
    pub emitter:          Chain,
    pub endpoint_address: [u8; 32],
}

impl GovernanceAction for GovernanceRegisterChain {
    const MODULE: &'static [u8] = b"TokenBridge";
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
    const MODULE: &'static [u8] = b"TokenBridge";
    const ACTION: u8 = 2;
    fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, new_contract) = parse_fixed(input)?;
        Ok((i, Self { new_contract }))
    }
}
