//! VAA's represent a collection of signatures combined with a message and its metadata. VAA's are
//! used as a form of proof; by submitting a VAA to a target contract, the receiving contract can
//! make assumptions about the validity of state on the source chain.
//!
//! Wormhole defines several VAA's for use within Token/NFT bridge implemenetations, as well as
//! governance specific VAA's used within Wormhole's guardian network.
//!
//! This module provides definitions and parsers for all current Wormhole standard VAA's, and
//! includes parsers for the core VAA type. Programs targetting wormhole can use this module to
//! parse and verify incoming VAA's securely.

use nom::combinator::rest;
use nom::error::{
    Error,
    ErrorKind,
};
use nom::multi::{
    count,
    fill,
};
use nom::number::complete::{
    u16,
    u32,
    u64,
    u8,
};
use nom::number::Endianness;
use nom::{
    Err,
    Finish,
    IResult,
};
use std::convert::TryFrom;
use std::str::FromStr; // Remove in 2021

use crate::WormholeError::{
    InvalidGovernanceAction,
    InvalidGovernanceChain,
    InvalidGovernanceModule,
};
use crate::{
    require,
    Chain,
    WormholeError,
};

// Import Module Specific VAAs.

pub mod core;
pub mod nft;
pub mod token;


/// Signatures are typical ECDSA signatures prefixed with a Guardian position. These have the
/// following byte layout:
/// ```markdown
/// 0  ..  1: Guardian No.
/// 1  .. 65: Signature   (ECDSA)
/// 65 .. 66: Recovery ID (ECDSA)
/// ```
pub type Signature = [u8; 66];

/// Wormhole specifies token addresses as 32 bytes. Addresses that are shorter, for example 20 byte
/// Ethereum addresses, are left zero padded to 32.
pub type ForeignAddress = [u8; 32];

/// Fields on VAA's are all usually fixed bytestrings, however they often contain UTF-8. When
/// parsed these result in `String` with the additional constraint that they are always equal or
/// less to the underlying byte field.
type ShortUTFString = String;

/// The core VAA itself. This structure is what is received by a contract on the receiving side of
/// a wormhole message passing flow. The payload of the message must be parsed separately to the
/// VAA itself as it is completely user defined.
#[derive(Debug, Default, PartialEq)]
pub struct VAA {
    // Header
    pub version:            u8,
    pub guardian_set_index: u32,
    pub signatures:         Vec<Signature>,

    // Body
    pub timestamp:         u32,
    pub nonce:             u32,
    pub emitter_chain:     Chain,
    pub emitter_address:   ForeignAddress,
    pub sequence:          u64,
    pub consistency_level: u8,
    pub payload:           Vec<u8>,
}

/// Contains the hash, secp256k1 payload, and serialized digest of the VAA. These are used in
/// various places in Wormhole codebases.
pub struct VAADigest {
    pub digest: Vec<u8>,
    pub hash:   [u8; 32],
}

impl VAA {
    /// Given any argument treatable as a series of bytes, attempt to deserialize into a valid VAA.
    pub fn from_bytes<T: AsRef<[u8]>>(input: T) -> Result<Self, WormholeError> {
        match parse_vaa(input.as_ref()).finish() {
            Ok(input) => Ok(input.1),
            Err(e) => Err(WormholeError::ParseError(e.code as usize)),
        }
    }

    /// A VAA is distinguished by the unique hash of its deterministic components. This method
    /// returns a 256 bit Keccak hash of these components. This hash is utilised in all Wormhole
    /// components for identifying unique VAA's, including the bridge, modules, and core guardian
    /// software.
    pub fn digest(&self) -> Option<VAADigest> {
        use byteorder::{
            BigEndian,
            WriteBytesExt,
        };
        use sha3::Digest;
        use std::io::{
            Cursor,
            Write,
        };

        // Hash Deterministic Pieces
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(self.timestamp).ok()?;
            v.write_u32::<BigEndian>(self.nonce).ok()?;
            v.write_u16::<BigEndian>(self.emitter_chain.clone() as u16).ok()?;
            let _ = v.write(&self.emitter_address).ok()?;
            v.write_u64::<BigEndian>(self.sequence).ok()?;
            v.write_u8(self.consistency_level).ok()?;
            let _ = v.write(&self.payload).ok()?;
            v.into_inner()
        };

        // We hash the body so that secp256k1 signatures are signing the hash instead of the body
        // within our contracts. We do this so we don't have to submit the entire VAA for signature
        // verification, only the hash.
        let hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            let _ = h.write(body.as_slice()).unwrap();
            h.finalize().into()
        };

        Some(VAADigest {
            digest: body,
            hash,
        })
    }
}

/// Using nom, parse a fixed array of bytes without any allocation. Useful for parsing addresses,
/// signatures, identifiers, etc.
#[inline]
pub fn parse_fixed<const S: usize>(input: &[u8]) -> IResult<&[u8], [u8; S]> {
    let mut buffer = [0u8; S];
    let (i, _) = fill(u8, &mut buffer)(input)?;
    Ok((i, buffer))
}

/// Parse a Chain ID, which is a 16 bit numeric ID. The mapping of network to ID is defined by the
/// Wormhole standard.
#[inline]
pub fn parse_chain(input: &[u8]) -> IResult<&[u8], Chain> {
    let (i, chain) = u16(Endianness::Big)(input)?;
    let chain = Chain::try_from(chain).map_err(|_| Err::Error(Error::new(i, ErrorKind::NoneOf)))?;
    Ok((i, chain))
}

/// Parse a VAA from a vector of raw bytes. Nom handles situations where the data is either too
/// short or too long.
#[inline]
fn parse_vaa(input: &[u8]) -> IResult<&[u8], VAA> {
    let (i, version) = u8(input)?;
    let (i, guardian_set_index) = u32(Endianness::Big)(i)?;
    let (i, signature_count) = u8(i)?;
    let (i, signatures) = count(parse_fixed, signature_count.into())(i)?;
    let (i, timestamp) = u32(Endianness::Big)(i)?;
    let (i, nonce) = u32(Endianness::Big)(i)?;
    let (i, emitter_chain) = parse_chain(i)?;
    let (i, emitter_address) = parse_fixed(i)?;
    let (i, sequence) = u64(Endianness::Big)(i)?;
    let (i, consistency_level) = u8(i)?;
    let (i, payload) = rest(i)?;
    Ok((
        i,
        VAA {
            version,
            guardian_set_index,
            signatures,
            timestamp,
            nonce,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            payload: payload.to_vec(),
        },
    ))
}

/// All current Wormhole programs using Governance are prefixed with a Governance header with a
/// consistent format.
pub struct GovHeader {
    pub module: [u8; 32],
    pub action: u8,
    pub chains: Chain,
}

pub trait GovernanceAction: Sized {
    const ACTION: u8;
    const MODULE: &'static [u8];

    /// Implement a nom parser for the Action.
    fn parse(input: &[u8]) -> IResult<&[u8], Self>;

    /// Serialize to Wormhole wire format.
    /// fn serialize(&self) -> Result<Vec<u8>, WormholeError>;

    /// Parses an Action from a governance payload securely.
    fn from_bytes<T: AsRef<[u8]>>(
        input: T,
        chain: Option<Chain>,
    ) -> Result<(GovHeader, Self), WormholeError> {
        match parse_action(input.as_ref()).finish() {
            Ok((_, (header, action))) => {
                // If no Chain is given, we assume All, which implies always valid.
                let chain = chain.unwrap_or(Chain::All);

                // Left 0-pad the MODULE in case it is unpadded.
                let mut module = [0u8; 32];
                let modlen = Self::MODULE.len();
                (&mut module[32 - modlen..]).copy_from_slice(&Self::MODULE);

                // Verify Governance Data.
                let valid_chain = chain == header.chains || chain == Chain::All;
                let valid_action = header.action == Self::ACTION;
                let valid_module = module == header.module;
                require!(valid_action, InvalidGovernanceAction);
                require!(valid_chain, InvalidGovernanceChain);
                require!(valid_module, InvalidGovernanceModule);

                Ok((header, action))
            }
            Err(e) => Err(WormholeError::ParseError(e.code as usize)),
        }
    }
}

#[inline]
pub fn parse_action<A: GovernanceAction>(input: &[u8]) -> IResult<&[u8], (GovHeader, A)> {
    let (i, header) = parse_governance_header(input.as_ref())?;
    let (i, action) = A::parse(i)?;
    Ok((i, (header, action)))
}

#[inline]
pub fn parse_governance_header<'i, 'a>(input: &'i [u8]) -> IResult<&'i [u8], GovHeader> {
    let (i, module) = parse_fixed(input)?;
    let (i, action) = u8(i)?;
    let (i, chains) = u16(Endianness::Big)(i)?;
    Ok((
        i,
        GovHeader {
            module,
            action,
            chains: Chain::try_from(chains).unwrap(),
        },
    ))
}

#[cfg(test)]
mod testing {
    use super::{
        parse_governance_header,
        Chain,
        VAA,
    };

    #[test]
    fn test_valid_gov_header() {
        let signers = hex::decode("00b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e9801").unwrap();
        let payload = hex::decode("000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let emitter =
            hex::decode("0000000000000000000000000000000000000000000000000000000000000004")
                .unwrap();
        let module =
            hex::decode("000000000000000000000000000000000000000000546f6b656e427269646765")
                .unwrap();

        // Decode VAA.
        let vaa = hex::decode("01000000000100b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e980100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000013c1bfa00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let vaa = VAA::from_bytes(vaa).unwrap();

        // Decode Payload
        let (_, header) = parse_governance_header(&vaa.payload).unwrap();

        // Confirm Parsed matches Required.
        assert_eq!(&header.module, &module[..]);
        assert_eq!(header.action, 1);
        assert_eq!(header.chains, Chain::All);
    }

    // Legacy VAA Signature Struct.
    #[derive(Default, Clone)]
    pub struct VAASignature {
        pub signature:      Vec<u8>,
        pub guardian_index: u8,
    }

    // Original VAA Parsing Code. Used to compare current code to old for parity.
    pub fn legacy_deserialize(data: &[u8]) -> std::result::Result<VAA, std::io::Error> {
        use byteorder::{
            BigEndian,
            ReadBytesExt,
        };
        use std::convert::TryFrom;
        use std::io::Read;

        let mut rdr = std::io::Cursor::new(data);
        let mut v = VAA {
            ..Default::default()
        };
        v.version = rdr.read_u8()?;
        v.guardian_set_index = rdr.read_u32::<BigEndian>()?;
        let len_sig = rdr.read_u8()?;
        let mut sigs: Vec<_> = Vec::with_capacity(len_sig as usize);
        for _i in 0..len_sig {
            let mut sig = [0u8; 66];
            sig[0] = rdr.read_u8()?;
            rdr.read_exact(&mut sig[1..66])?;
            sigs.push(sig);
        }
        v.signatures = sigs;
        v.timestamp = rdr.read_u32::<BigEndian>()?;
        v.nonce = rdr.read_u32::<BigEndian>()?;
        v.emitter_chain = Chain::try_from(rdr.read_u16::<BigEndian>()?).unwrap();
        let mut emitter_address = [0u8; 32];
        rdr.read_exact(&mut emitter_address)?;
        v.emitter_address = emitter_address;
        v.sequence = rdr.read_u64::<BigEndian>()?;
        v.consistency_level = rdr.read_u8()?;
        let _ = rdr.read_to_end(&mut v.payload)?;
        Ok(v)
    }

    #[test]
    fn test_parse_vaa_parity() {
        // Decode VAA with old and new parsers, and compare result.
        let vaa = hex::decode("01000000000100b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e980100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000013c1bfa00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let new = VAA::from_bytes(&vaa).unwrap();
        let old = legacy_deserialize(&vaa).unwrap();
        assert_eq!(new, old);
    }

    #[test]
    fn test_valid_parse_vaa() {
        let signers = hex::decode("00b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e9801").unwrap();
        let payload = hex::decode("000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let emitter =
            hex::decode("0000000000000000000000000000000000000000000000000000000000000004")
                .unwrap();

        // Decode VAA.
        let vaa = hex::decode("01000000000100b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e980100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000013c1bfa00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let vaa = VAA::from_bytes(vaa).unwrap();

        // Verify Decoded VAA.
        assert_eq!(vaa.version, 1);
        assert_eq!(vaa.guardian_set_index, 0);
        assert_eq!(vaa.signatures.len(), 1);
        assert_eq!(vaa.signatures[0][..], signers);
        assert_eq!(vaa.timestamp, 1);
        assert_eq!(vaa.nonce, 1);
        assert_eq!(vaa.emitter_chain, Chain::Solana);
        assert_eq!(vaa.emitter_address, emitter[..]);
        assert_eq!(vaa.sequence, 20_716_538);
        assert_eq!(vaa.consistency_level, 0);
        assert_eq!(vaa.payload, payload);
    }

    #[test]
    fn test_invalid_vaa() {
    }
}
