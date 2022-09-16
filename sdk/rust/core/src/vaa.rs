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

use {
    crate::{
        parse_chain,
        parse_fixed,
        Chain,
        Error,
    },
    nom::{
        combinator::{
            flat_map,
            rest,
        },
        multi::count,
        number::{
            complete::{
                u32,
                u64,
                u8,
            },
            Endianness,
        },
        IResult,
    },
    std::result::Result,
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
#[derive(Clone, Debug, Default, PartialEq, Eq)]
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

/// VAADigest contains useful digest data for the VAA.
///
/// - The Digest itself.
/// - A hash of the Digest, which is what a guardian actually signs.
/// - The secp256k1 message part,  ahash of the hash of the Digest, which can be passed to ecrecover.
pub struct VAADigest {
    pub digest:  Vec<u8>,
    pub hash:    [u8; 32],
    pub message: [u8; 32],
}

impl VAA {
    /// Given a series of bytes, attempt to deserialize into a valid VAA. Nom handles situations
    /// where the data is either too short or too long.
    pub fn from_bytes<T: AsRef<[u8]>>(i: T) -> Result<Self, Error> {
        let (
            _,
            (
                version,
                guardian_set_index,
                signatures,
                timestamp,
                nonce,
                emitter_chain,
                emitter_address,
                sequence,
                consistency_level,
                payload,
            ),
        ) = nom::sequence::tuple((
            u8,
            u32(Endianness::Big),
            flat_map(u8, |c| count(parse_fixed, c.into())),
            u32(Endianness::Big),
            u32(Endianness::Big),
            parse_chain,
            parse_fixed,
            u64(Endianness::Big),
            u8,
            rest,
        ))(i.as_ref())?;

        Ok(VAA {
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
        })
    }

    /// Check if the VAA is a Governance VAA.
    pub fn is_governance(&self) -> bool {
        self.emitter_address == crate::GOVERNANCE_EMITTER && self.emitter_chain == Chain::Solana
    }

    /// Extract the body of the VAA, used as the payload for the digest.
    pub fn body(&self) -> Result<Vec<u8>, Error> {
        use {
            byteorder::{
                BigEndian,
                WriteBytesExt,
            },
            std::io::{
                Cursor,
                Write,
            },
        };

        let mut v = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(self.timestamp)?;
        v.write_u32::<BigEndian>(self.nonce)?;
        v.write_u16::<BigEndian>(self.emitter_chain.into())?;
        let _ = v.write(&self.emitter_address)?;
        v.write_u64::<BigEndian>(self.sequence)?;
        v.write_u8(self.consistency_level)?;
        let _ = v.write(&self.payload)?;
        Ok(v.into_inner())
    }

    /// VAA Digest Components.
    ///
    /// A VAA is distinguished by the unique 256bit Keccak256 hash of its body. This hash is
    /// utilised in all Wormhole components for identifying unique VAA's, including the bridge,
    /// modules, and core guardian software. See `VAADigest` for more information.
    ///
    /// If efficiency is required, hashing using on-chain built-ins instead is recommended, this
    /// can be done by calling `body` and hashing manually.
    pub fn digest(&self) -> Option<VAADigest> {
        use {
            sha3::Digest,
            std::io::Write,
        };

        // We hash the body so that secp256k1 signatures are signing the hash instead of the body
        // within our contracts. We do this so we don't have to submit the entire VAA for signature
        // verification, only the hash.
        let hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            let _ = h.write(body.as_slice()).unwrap();
            h.finalize().into()
        };

        // TODO: Explain double hashing reason.
        let message: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            let _ = h.write(&hash).unwrap();
            h.finalize().into()
        };

        Some(VAADigest {
            digest: body,
            hash,
            message,
        })
    }
}

/// All current Wormhole programs using Governance are prefixed with a Governance header.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GovHeader {
    pub module: [u8; 32],
    pub action: u8,
    pub target: Chain,
}

impl GovHeader {
    // Given a Chain and Module, produce a parser for a GovHeader.
    pub fn parse(input: &[u8]) -> IResult<&[u8], Self> {
        let (i, (module, action, target)) =
            nom::sequence::tuple((parse_fixed, u8, parse_chain))(input)?;

        Ok((
            i,
            GovHeader {
                module,
                action,
                target,
            },
        ))
    }
}

/// The Action trait describes functionality that various VAA payload formats used by Wormhole
/// applications confirm to.
pub trait Action: Sized {
    /// Parse an action from a VAA.
    fn from_vaa(vaa: &VAA, chain: Chain) -> Result<Self, Error>;
}
