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
        Chain,
        WormholeError,
    },
    nom::{
        combinator::{
            flat_map,
            rest,
        },
        error::ErrorKind,
        multi::{
            count,
            fill,
        },
        number::{
            complete::{
                u16,
                u32,
                u64,
                u8,
            },
            Endianness,
        },
        IResult,
    },
    std::convert::TryFrom,
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
    pub fn from_bytes<T: AsRef<[u8]>>(i: T) -> Result<Self, WormholeError> {
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

    /// VAA Digest Components.
    ///
    /// A VAA is distinguished by the unique hash of its deterministic components. This method
    /// returns a 256 bit Keccak hash of these components. This hash is utilised in all Wormhole
    /// components for identifying unique VAA's, including the bridge, modules, and core guardian
    /// software. See `VAADigest` for more information.
    pub fn digest(&self) -> Option<VAADigest> {
        use {
            byteorder::{
                BigEndian,
                WriteBytesExt,
            },
            sha3::Digest,
            std::io::{
                Cursor,
                Write,
            },
        };

        // Hash Deterministic Pieces
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(self.timestamp).ok()?;
            v.write_u32::<BigEndian>(self.nonce).ok()?;
            v.write_u16::<BigEndian>(self.emitter_chain.into()).ok()?;
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

        // We also hash the hash so we can provide SDK users with the secp256k1 message part, which
        // is useful if a contract wants to use ecrecover.
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
    Ok((
        i,
        Chain::try_from(chain)
            .map_err(|_| nom::Err::Error(nom::error_position!(i, ErrorKind::NoneOf)))?,
    ))
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
    fn from_vaa(vaa: &VAA, chain: Chain) -> Result<Self, WormholeError>;
}

#[cfg(test)]
mod testing {
    use {
        super::*,
        crate::error::WormholeError,
    };

    #[test]
    fn test_valid_gov_header() {
        let module =
            hex::decode("000000000000000000000000000000000000000000546f6b656e427269646765")
                .unwrap();

        // Decode VAA.
        let vaa = hex::decode("01000000000100b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e980100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000013c1bfa00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let vaa = VAA::from_bytes(vaa).unwrap();

        // Decode Payload
        let (_, header) = GovHeader::parse(&vaa.payload).unwrap();

        // Confirm Parsed matches Required.
        assert_eq!(&header.module, &module[..]);
        assert_eq!(header.action, 1);
        assert_eq!(header.target, Chain::Any);
    }

    // Original VAA Parsing Code. Used to compare current code to old for parity.
    pub fn legacy_deserialize(data: &[u8]) -> std::result::Result<VAA, std::io::Error> {
        use {
            byteorder::{
                BigEndian,
                ReadBytesExt,
            },
            std::io::Read,
        };

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

    /// Deserializes a VAA using both the old and new parser and confirms they're equivalent.
    #[test]
    fn test_parse_vaa_parity() {
        // Decode VAA with old and new parsers, and compare result.
        let vaa = hex::decode("01000000000100b072505b5b999c1d08905c02e2b6b2832ef72c0ba6c8db4f77fe457ef2b3d053410b1e92a9194d9210df24d987ac83d7b6f0c21ce90f8bc1869de0898bda7e980100000001000000010001000000000000000000000000000000000000000000000000000000000000000400000000013c1bfa00000000000000000000000000000000000000000000546f6b656e42726964676501000000013b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98").unwrap();
        let new = VAA::from_bytes(&vaa).unwrap();
        let old = legacy_deserialize(&vaa).unwrap();
        assert_eq!(new, old);
    }

    /// Checks an arbitrary VAA against what we expect parsing it should be.
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

    /// Checks VAA parser errors if not enough data present.
    #[test]
    fn test_short_vaa() {
        // Too Short Input.
        let vaa = hex::decode("01000000").unwrap();
        let vaa = VAA::from_bytes(vaa);
        assert!(matches!(vaa.unwrap_err(), WormholeError::ParseError(_)));
    }

    #[test]
    fn test_base64_expected1() {
        let vaa = base64::decode("AQAAAAENAnOvPC9VmJBBsAaKTq66j4glEGAhmW3mFDlUwG/Ez61cPxq6bKlRGI6WLxElHCXKmGKGheL8K2XsYJRQz8TmURIAAwDGLrDeerPQKJIPGOEN9/KgTQDriLiUCie7zozmbGdSaILJbMUN04G9v/MLtluTR8rf8JZ2cpBDr2DWUqC5BjAABPB7D+bKsYvnroJ/4RyomS/wtaKjLWW+lYIxv4TPaxT7XuuKUa3hxwqluLjPg6/jwi00cUgb2jiW6ipwRp+WkrgABXTUlnKd3m4ZCVmheUXofNleI8EAR6su71x9Dsb5EgjHJ52KGx9KYAadJZMqZ9ZV8tC0IFkAPedf08p5kv3RsNQBBlHwarb9/ULzI4QKgYs4z9HJnSI2bId5A7mN9Ava8qIELrjNDlnEY35qgKGZsRCM12WbqDcPb5R2tHmDmFTYwaYAB3hLN8YQPHUs2XpYa+jhzv8ipuSIQzKE/zHNkItcfYfiRNp1FtB6D6aSaE+Cbl5si0UgBCBtb+W65Gr7HCGM9Q0ACQtszOZ+1QHLIPsG3na5CD8TKa1404RRepSrjpqmAb56DwC7YDs2UEp03cNnNZyOoH9czVAidyzBV+APVBVjceQBC+HmtxKiNT5JB5KcQFfVur74DcCf67PcKTT0QEh5Xu+VTpkQbLKbGo2TU2na7LuLrkUZLvw87bxXMV1n7J6oAAoADdzATNdVapTotBjcOooA77Eo1PdvcUMSR6kuehmoM/wCIV0f1p4OWW2lMepYeuKsLzbSDzsZMwYK1u8+nX2EdboBDsdiFklJBq7Y2DEMMaXkpUXqKvjb447rdKPRTwc03SsaIbmFDqIObCykIkh4i/sXQ503q9ol1wW1aLJlsRO5dsUAEMYf5uvqYfLK6JXDNJZQcEh9Oatr8EQoNArw92mf3dPAHgyG2uqetElwEkiiT6TA/3X7YAssATS9cheR9mcRbkIAERm3nKolDnXaFILH9BJocwjRPvcA9ya5lBe7da5t0UP8YZ+MnCnHP5lJr/0WKcwoamGRfteFN1SkZwKSC3bPkOsAErC6teGosyHpL509+SQwGD5IQ+V8b4wVwi1isvjkuM3CdZ8oVLjRIqzbS02JKLh99BmcxoMBHS6bfUGpbYtIDMwBAAAAADnEunYAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEJWTS79YITfAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAVG9rZW5CcmlkZ2UCAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJgw==").unwrap();
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 1,
            signatures:         vec![
                hex_literal::hex!("0273af3c2f55989041b0068a4eaeba8f8825106021996de6143954c06fc4cfad5c3f1aba6ca951188e962f11251c25ca98628685e2fc2b65ec609450cfc4e6511200"),
                hex_literal::hex!("0300c62eb0de7ab3d028920f18e10df7f2a04d00eb88b8940a27bbce8ce66c67526882c96cc50dd381bdbff30bb65b9347cadff09676729043af60d652a0b9063000"),
                hex_literal::hex!("04f07b0fe6cab18be7ae827fe11ca8992ff0b5a2a32d65be958231bf84cf6b14fb5eeb8a51ade1c70aa5b8b8cf83afe3c22d3471481bda3896ea2a70469f9692b800"),
                hex_literal::hex!("0574d496729dde6e190959a17945e87cd95e23c10047ab2eef5c7d0ec6f91208c7279d8a1b1f4a60069d25932a67d655f2d0b42059003de75fd3ca7992fdd1b0d401"),
                hex_literal::hex!("0651f06ab6fdfd42f323840a818b38cfd1c99d22366c877903b98df40bdaf2a2042eb8cd0e59c4637e6a80a199b1108cd7659ba8370f6f9476b479839854d8c1a600"),
                hex_literal::hex!("07784b37c6103c752cd97a586be8e1ceff22a6e488433284ff31cd908b5c7d87e244da7516d07a0fa692684f826e5e6c8b452004206d6fe5bae46afb1c218cf50d00"),
                hex_literal::hex!("090b6ccce67ed501cb20fb06de76b9083f1329ad78d384517a94ab8e9aa601be7a0f00bb603b36504a74ddc367359c8ea07f5ccd5022772cc157e00f54156371e401"),
                hex_literal::hex!("0be1e6b712a2353e4907929c4057d5babef80dc09febb3dc2934f44048795eef954e99106cb29b1a8d935369daecbb8bae45192efc3cedbc57315d67ec9ea8000a00"),
                hex_literal::hex!("0ddcc04cd7556a94e8b418dc3a8a00efb128d4f76f71431247a92e7a19a833fc02215d1fd69e0e596da531ea587ae2ac2f36d20f3b1933060ad6ef3e9d7d8475ba01"),
                hex_literal::hex!("0ec76216494906aed8d8310c31a5e4a545ea2af8dbe38eeb74a3d14f0734dd2b1a21b9850ea20e6c2ca42248788bfb17439d37abda25d705b568b265b113b976c500"),
                hex_literal::hex!("10c61fe6ebea61f2cae895c334965070487d39ab6bf04428340af0f7699fddd3c01e0c86daea9eb449701248a24fa4c0ff75fb600b2c0134bd721791f667116e4200"),
                hex_literal::hex!("1119b79caa250e75da1482c7f412687308d13ef700f726b99417bb75ae6dd143fc619f8c9c29c73f9949affd1629cc286a61917ed7853754a46702920b76cf90eb00"),
                hex_literal::hex!("12b0bab5e1a8b321e92f9d3df92430183e4843e57c6f8c15c22d62b2f8e4b8cdc2759f2854b8d122acdb4b4d8928b87df4199cc683011d2e9b7d41a96d8b480ccc01"),
            ],
            timestamp:          0,
            nonce:              969194102,
            emitter_chain:      1.into(),
            emitter_address:    hex::decode(
                "0000000000000000000000000000000000000000000000000000000000000004"
            )
            .unwrap()
            .try_into()
            .unwrap(),
            sequence:           2694510404604284400,
            consistency_level:  32,
            payload:            vec![0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 84, 111, 107, 101, 110, 66, 114, 105, 100, 103, 101, 2, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 131],
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            token::Action::from_vaa(&vaa, Chain::TerraClassic).unwrap(),
            token::Action::ContractUpgrade(token::ContractUpgrade {
                new_contract: hex_literal::hex!(
                    "0000000000000000000000000000000000000000000000000000000000000983"
                ),
            }),
        );

        // Catch-All
        assert_eq!(vaa, val);
    }

    /// Check arbitrary payload works correctly.
    #[test]
    fn test_base64_expected2() {
        let vaa = base64::decode("AQAAAAABAOKdOtGAsVPWjD9EXXXqpi/MmWkJRbqvStBGPpzkTyf3XaPUn3lyKSCqyBuivoD2iIlfF0lC/txAO8TlzjVVt3sAYrn3kQAAAAAAAgAAAAAAAAAAAAAAAPGaKgG3BRn2etswmplOyMaaln6LAAAAAAAAAAABRnJvbTogZXZtMFxuTXNnOiBIZWxsbyBXb3JsZCE=").unwrap();
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!(
                    "00e29d3ad180b153d68c3f445d75eaa62fcc99690945baaf4ad0463e9ce44f27f75da3d49f79722920aac81ba2be80f688895f174942fedc403bc4e5ce3555b77b00"
                ),
            ],
            timestamp:          1656354705,
            nonce:              0,
            emitter_chain:      2.into(),
            emitter_address:    hex_literal::hex!(
                "000000000000000000000000f19a2a01b70519f67adb309a994ec8c69a967e8b"
            ),
            sequence:           0,
            consistency_level:  1,
            payload:            vec![70, 114, 111, 109, 58, 32, 101, 118, 109, 48, 92, 110, 77, 115, 103, 58, 32, 72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100, 33],
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(vaa.payload, val.payload);

        // Catch-All
        assert_eq!(vaa, val);
    }

    #[test]
    fn test_big_payload() {
        let vaa = &hex_literal::hex!("01000000000100fd4cdd0e5a1afd9eb6555770fb132bf03ed8fa1f9e92c6adcec7881ace2ba4ba4c1b350f79da4110d3307053ceb217e4398eaf02be5474a90bd694b0d2ccbdcc0100000000baa551d500010000000000000000000000000000000000000000000000000000000000000004a3fff7bcbfc4b4ac200300000000000000000000000000000000000000000000000000000000000f4240165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa30100010000000000000000000000007c4dfd6be62406e7f5a05eec96300da4048e70ff0002000000000000000000000000000000000000000000000000000000000000000000000000000005de4c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e73656374657475722061646970697363696e6720656c69742e204375726162697475722074656d7075732c206e6571756520656765742068656e64726572697420626962656e64756d2c20616e746520616e7465206469676e697373696d2065782c207175697320616363756d73616e20656c6974206175677565206e6563206c656f2e2050726f696e207669746165206a7573746f207669746165206c6163757320706f737565726520706f72747469746f722e204d61757269732073656420736167697474697320697073756d2e204d6f726269206d61737361206d61676e612c20706f7375657265206e6f6e20696163756c697320656765742c20756c74726963696573206174206c6967756c612e20446f6e656320756c74726963696573206e697369206573742c206574206c6f626f727469732073656d2073616769747469732073697420616d65742e20446f6e6563206665756769617420646f6c6f722061206f64696f2064696374756d2c20736564206c616f72656574206d61676e6120656765737461732e205175697371756520756c7472696369657320666163696c69736973206172637520617420616363756d73616e2e20496e20696163756c697320617420707572757320696e207472697374697175652e204d616563656e617320706f72747469746f722c206e69736c20612073656d706572206d616c6573756164612c2074656c6c7573206e65717565206d616c657375616461206c656f2c2071756973206d6f6c65737469652066656c6973206e69626820696e2065726f732e20446f6e656320766976657272612061726375206e6563206e756e63207072657469756d2c206567657420756c6c616d636f7270657220707572757320706f73756572652e2053757370656e646973736520706f74656e74692e204e616d2067726176696461206c656f206e6563207175616d2074696e636964756e7420766976657272612e205072616573656e74206163207375736369706974206f7263692e20566976616d757320736f64616c6573206d6178696d757320626c616e6469742e2050656c6c656e74657371756520696d706572646965742075726e61206174206e756e63206d616c6573756164612c20696e20617563746f72206d6173736120616c697175616d2e2050656c6c656e746573717565207363656c6572697371756520657569736d6f64206f64696f20612074656d706f722e204e756c6c612073656420706f7274612070757275732c20657520706f727461206f64696f2e20457469616d207175697320706c616365726174206e756c6c612e204e756e6320696e20636f6d6d6f646f206d692c20657520736f64616c6573206e756e632e20416c697175616d206c7563747573206c6f72656d2065742074696e636964756e74206c6163696e69612e20447569732076656c20697073756d206e69736c2e205072616573656e7420636f6e76616c6c697320656c6974206c6967756c612c206e656320706f72746120657374206d6178696d75732061632e204e756c6c61207072657469756d206c696265726f206567657420616e746520756c6c616d636f72706572206d61747469732e204e756c6c616d20766f6c75747061742c2074656c6c757320736564207363656c65726973717565206566666963697475722c206e69736c2061756775652070686172657472612066656c69732c2076656c2067726176696461206d61676e612075726e6120736564207175616d2e2044756973206964207072657469756d206475692e20496e74656765722072686f6e637573206d6174746973206a7573746f20612068656e6472657269742e20467573636520646f6c6f72206d61676e612c20706f72747469746f7220616320707572757320736f64616c65732c20657569736d6f6420766573746962756c756d20746f72746f722e20416c697175616d2070686172657472612065726174206a7573746f2c20696e20756c6c616d636f72706572207175616d2e");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!(
                    "00fd4cdd0e5a1afd9eb6555770fb132bf03ed8fa1f9e92c6adcec7881ace2ba4ba4c1b350f79da4110d3307053ceb217e4398eaf02be5474a90bd694b0d2ccbdcc01"
                ),
            ],
            timestamp:          0,
            nonce:              3131396565,
            emitter_chain:      1.into(),
            emitter_address:    hex_literal::hex!(
                "0000000000000000000000000000000000000000000000000000000000000004"
            ),
            sequence:           11817436337286722732,
            consistency_level:  32,
            payload:            vec![3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 15, 66, 64, 22, 88, 9, 115, 146, 64, 160, 172, 3, 185, 132, 64, 254, 137, 133, 84, 142, 58, 166, 131, 205, 13, 77, 157, 245, 181, 101, 150, 105, 250, 163, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 124, 77, 253, 107, 230, 36, 6, 231, 245, 160, 94, 236, 150, 48, 13, 164, 4, 142, 112, 255, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 222, 76, 111, 114, 101, 109, 32, 105, 112, 115, 117, 109, 32, 100, 111, 108, 111, 114, 32, 115, 105, 116, 32, 97, 109, 101, 116, 44, 32, 99, 111, 110, 115, 101, 99, 116, 101, 116, 117, 114, 32, 97, 100, 105, 112, 105, 115, 99, 105, 110, 103, 32, 101, 108, 105, 116, 46, 32, 67, 117, 114, 97, 98, 105, 116, 117, 114, 32, 116, 101, 109, 112, 117, 115, 44, 32, 110, 101, 113, 117, 101, 32, 101, 103, 101, 116, 32, 104, 101, 110, 100, 114, 101, 114, 105, 116, 32, 98, 105, 98, 101, 110, 100, 117, 109, 44, 32, 97, 110, 116, 101, 32, 97, 110, 116, 101, 32, 100, 105, 103, 110, 105, 115, 115, 105, 109, 32, 101, 120, 44, 32, 113, 117, 105, 115, 32, 97, 99, 99, 117, 109, 115, 97, 110, 32, 101, 108, 105, 116, 32, 97, 117, 103, 117, 101, 32, 110, 101, 99, 32, 108, 101, 111, 46, 32, 80, 114, 111, 105, 110, 32, 118, 105, 116, 97, 101, 32, 106, 117, 115, 116, 111, 32, 118, 105, 116, 97, 101, 32, 108, 97, 99, 117, 115, 32, 112, 111, 115, 117, 101, 114, 101, 32, 112, 111, 114, 116, 116, 105, 116, 111, 114, 46, 32, 77, 97, 117, 114, 105, 115, 32, 115, 101, 100, 32, 115, 97, 103, 105, 116, 116, 105, 115, 32, 105, 112, 115, 117, 109, 46, 32, 77, 111, 114, 98, 105, 32, 109, 97, 115, 115, 97, 32, 109, 97, 103, 110, 97, 44, 32, 112, 111, 115, 117, 101, 114, 101, 32, 110, 111, 110, 32, 105, 97, 99, 117, 108, 105, 115, 32, 101, 103, 101, 116, 44, 32, 117, 108, 116, 114, 105, 99, 105, 101, 115, 32, 97, 116, 32, 108, 105, 103, 117, 108, 97, 46, 32, 68, 111, 110, 101, 99, 32, 117, 108, 116, 114, 105, 99, 105, 101, 115, 32, 110, 105, 115, 105, 32, 101, 115, 116, 44, 32, 101, 116, 32, 108, 111, 98, 111, 114, 116, 105, 115, 32, 115, 101, 109, 32, 115, 97, 103, 105, 116, 116, 105, 115, 32, 115, 105, 116, 32, 97, 109, 101, 116, 46, 32, 68, 111, 110, 101, 99, 32, 102, 101, 117, 103, 105, 97, 116, 32, 100, 111, 108, 111, 114, 32, 97, 32, 111, 100, 105, 111, 32, 100, 105, 99, 116, 117, 109, 44, 32, 115, 101, 100, 32, 108, 97, 111, 114, 101, 101, 116, 32, 109, 97, 103, 110, 97, 32, 101, 103, 101, 115, 116, 97, 115, 46, 32, 81, 117, 105, 115, 113, 117, 101, 32, 117, 108, 116, 114, 105, 99, 105, 101, 115, 32, 102, 97, 99, 105, 108, 105, 115, 105, 115, 32, 97, 114, 99, 117, 32, 97, 116, 32, 97, 99, 99, 117, 109, 115, 97, 110, 46, 32, 73, 110, 32, 105, 97, 99, 117, 108, 105, 115, 32, 97, 116, 32, 112, 117, 114, 117, 115, 32, 105, 110, 32, 116, 114, 105, 115, 116, 105, 113, 117, 101, 46, 32, 77, 97, 101, 99, 101, 110, 97, 115, 32, 112, 111, 114, 116, 116, 105, 116, 111, 114, 44, 32, 110, 105, 115, 108, 32, 97, 32, 115, 101, 109, 112, 101, 114, 32, 109, 97, 108, 101, 115, 117, 97, 100, 97, 44, 32, 116, 101, 108, 108, 117, 115, 32, 110, 101, 113, 117, 101, 32, 109, 97, 108, 101, 115, 117, 97, 100, 97, 32, 108, 101, 111, 44, 32, 113, 117, 105, 115, 32, 109, 111, 108, 101, 115, 116, 105, 101, 32, 102, 101, 108, 105, 115, 32, 110, 105, 98, 104, 32, 105, 110, 32, 101, 114, 111, 115, 46, 32, 68, 111, 110, 101, 99, 32, 118, 105, 118, 101, 114, 114, 97, 32, 97, 114, 99, 117, 32, 110, 101, 99, 32, 110, 117, 110, 99, 32, 112, 114, 101, 116, 105, 117, 109, 44, 32, 101, 103, 101, 116, 32, 117, 108, 108, 97, 109, 99, 111, 114, 112, 101, 114, 32, 112, 117, 114, 117, 115, 32, 112, 111, 115, 117, 101, 114, 101, 46, 32, 83, 117, 115, 112, 101, 110, 100, 105, 115, 115, 101, 32, 112, 111, 116, 101, 110, 116, 105, 46, 32, 78, 97, 109, 32, 103, 114, 97, 118, 105, 100, 97, 32, 108, 101, 111, 32, 110, 101, 99, 32, 113, 117, 97, 109, 32, 116, 105, 110, 99, 105, 100, 117, 110, 116, 32, 118, 105, 118, 101, 114, 114, 97, 46, 32, 80, 114, 97, 101, 115, 101, 110, 116, 32, 97, 99, 32, 115, 117, 115, 99, 105, 112, 105, 116, 32, 111, 114, 99, 105, 46, 32, 86, 105, 118, 97, 109, 117, 115, 32, 115, 111, 100, 97, 108, 101, 115, 32, 109, 97, 120, 105, 109, 117, 115, 32, 98, 108, 97, 110, 100, 105, 116, 46, 32, 80, 101, 108, 108, 101, 110, 116, 101, 115, 113, 117, 101, 32, 105, 109, 112, 101, 114, 100, 105, 101, 116, 32, 117, 114, 110, 97, 32, 97, 116, 32, 110, 117, 110, 99, 32, 109, 97, 108, 101, 115, 117, 97, 100, 97, 44, 32, 105, 110, 32, 97, 117, 99, 116, 111, 114, 32, 109, 97, 115, 115, 97, 32, 97, 108, 105, 113, 117, 97, 109, 46, 32, 80, 101, 108, 108, 101, 110, 116, 101, 115, 113, 117, 101, 32, 115, 99, 101, 108, 101, 114, 105, 115, 113, 117, 101, 32, 101, 117, 105, 115, 109, 111, 100, 32, 111, 100, 105, 111, 32, 97, 32, 116, 101, 109, 112, 111, 114, 46, 32, 78, 117, 108, 108, 97, 32, 115, 101, 100, 32, 112, 111, 114, 116, 97, 32, 112, 117, 114, 117, 115, 44, 32, 101, 117, 32, 112, 111, 114, 116, 97, 32, 111, 100, 105, 111, 46, 32, 69, 116, 105, 97, 109, 32, 113, 117, 105, 115, 32, 112, 108, 97, 99, 101, 114, 97, 116, 32, 110, 117, 108, 108, 97, 46, 32, 78, 117, 110, 99, 32, 105, 110, 32, 99, 111, 109, 109, 111, 100, 111, 32, 109, 105, 44, 32, 101, 117, 32, 115, 111, 100, 97, 108, 101, 115, 32, 110, 117, 110, 99, 46, 32, 65, 108, 105, 113, 117, 97, 109, 32, 108, 117, 99, 116, 117, 115, 32, 108, 111, 114, 101, 109, 32, 101, 116, 32, 116, 105, 110, 99, 105, 100, 117, 110, 116, 32, 108, 97, 99, 105, 110, 105, 97, 46, 32, 68, 117, 105, 115, 32, 118, 101, 108, 32, 105, 112, 115, 117, 109, 32, 110, 105, 115, 108, 46, 32, 80, 114, 97, 101, 115, 101, 110, 116, 32, 99, 111, 110, 118, 97, 108, 108, 105, 115, 32, 101, 108, 105, 116, 32, 108, 105, 103, 117, 108, 97, 44, 32, 110, 101, 99, 32, 112, 111, 114, 116, 97, 32, 101, 115, 116, 32, 109, 97, 120, 105, 109, 117, 115, 32, 97, 99, 46, 32, 78, 117, 108, 108, 97, 32, 112, 114, 101, 116, 105, 117, 109, 32, 108, 105, 98, 101, 114, 111, 32, 101, 103, 101, 116, 32, 97, 110, 116, 101, 32, 117, 108, 108, 97, 109, 99, 111, 114, 112, 101, 114, 32, 109, 97, 116, 116, 105, 115, 46, 32, 78, 117, 108, 108, 97, 109, 32, 118, 111, 108, 117, 116, 112, 97, 116, 44, 32, 116, 101, 108, 108, 117, 115, 32, 115, 101, 100, 32, 115, 99, 101, 108, 101, 114, 105, 115, 113, 117, 101, 32, 101, 102, 102, 105, 99, 105, 116, 117, 114, 44, 32, 110, 105, 115, 108, 32, 97, 117, 103, 117, 101, 32, 112, 104, 97, 114, 101, 116, 114, 97, 32, 102, 101, 108, 105, 115, 44, 32, 118, 101, 108, 32, 103, 114, 97, 118, 105, 100, 97, 32, 109, 97, 103, 110, 97, 32, 117, 114, 110, 97, 32, 115, 101, 100, 32, 113, 117, 97, 109, 46, 32, 68, 117, 105, 115, 32, 105, 100, 32, 112, 114, 101, 116, 105, 117, 109, 32, 100, 117, 105, 46, 32, 73, 110, 116, 101, 103, 101, 114, 32, 114, 104, 111, 110, 99, 117, 115, 32, 109, 97, 116, 116, 105, 115, 32, 106, 117, 115, 116, 111, 32, 97, 32, 104, 101, 110, 100, 114, 101, 114, 105, 116, 46, 32, 70, 117, 115, 99, 101, 32, 100, 111, 108, 111, 114, 32, 109, 97, 103, 110, 97, 44, 32, 112, 111, 114, 116, 116, 105, 116, 111, 114, 32, 97, 99, 32, 112, 117, 114, 117, 115, 32, 115, 111, 100, 97, 108, 101, 115, 44, 32, 101, 117, 105, 115, 109, 111, 100, 32, 118, 101, 115, 116, 105, 98, 117, 108, 117, 109, 32, 116, 111, 114, 116, 111, 114, 46, 32, 65, 108, 105, 113, 117, 97, 109, 32, 112, 104, 97, 114, 101, 116, 114, 97, 32, 101, 114, 97, 116, 32, 106, 117, 115, 116, 111, 44, 32, 105, 110, 32, 117, 108, 108, 97, 109, 99, 111, 114, 112, 101, 114, 32, 113, 117, 97, 109, 46],
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            token::Action::from_vaa(&vaa, Chain::Any).unwrap(),
            token::Action::TransferWithPayload(token::TransferWithPayload {
                amount:        1000000.into(),
                token_chain:   Chain::Solana,
                to_chain:      Chain::Ethereum,
                from_address:  hex_literal::hex!(
                    "0000000000000000000000000000000000000000000000000000000000000000"
                ),
                token_address: hex_literal::hex!(
                    "165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa301"
                ),
                to:            hex_literal::hex!(
                    "0000000000000000000000007c4dfd6be62406e7f5a05eec96300da4048e70ff"
                ),
                payload:      hex_literal::hex!(
                    "00000000000005de4c6f72656d20697073756d20646f6c6f722073697420616d65742c20636f6e73656374657475722061646970697363696e6720656c69742e204375726162697475722074656d7075732c206e6571756520656765742068656e64726572697420626962656e64756d2c20616e746520616e7465206469676e697373696d2065782c207175697320616363756d73616e20656c6974206175677565206e6563206c656f2e2050726f696e207669746165206a7573746f207669746165206c6163757320706f737565726520706f72747469746f722e204d61757269732073656420736167697474697320697073756d2e204d6f726269206d61737361206d61676e612c20706f7375657265206e6f6e20696163756c697320656765742c20756c74726963696573206174206c6967756c612e20446f6e656320756c74726963696573206e697369206573742c206574206c6f626f727469732073656d2073616769747469732073697420616d65742e20446f6e6563206665756769617420646f6c6f722061206f64696f2064696374756d2c20736564206c616f72656574206d61676e6120656765737461732e205175697371756520756c7472696369657320666163696c69736973206172637520617420616363756d73616e2e20496e20696163756c697320617420707572757320696e207472697374697175652e204d616563656e617320706f72747469746f722c206e69736c20612073656d706572206d616c6573756164612c2074656c6c7573206e65717565206d616c657375616461206c656f2c2071756973206d6f6c65737469652066656c6973206e69626820696e2065726f732e20446f6e656320766976657272612061726375206e6563206e756e63207072657469756d2c206567657420756c6c616d636f7270657220707572757320706f73756572652e2053757370656e646973736520706f74656e74692e204e616d2067726176696461206c656f206e6563207175616d2074696e636964756e7420766976657272612e205072616573656e74206163207375736369706974206f7263692e20566976616d757320736f64616c6573206d6178696d757320626c616e6469742e2050656c6c656e74657371756520696d706572646965742075726e61206174206e756e63206d616c6573756164612c20696e20617563746f72206d6173736120616c697175616d2e2050656c6c656e746573717565207363656c6572697371756520657569736d6f64206f64696f20612074656d706f722e204e756c6c612073656420706f7274612070757275732c20657520706f727461206f64696f2e20457469616d207175697320706c616365726174206e756c6c612e204e756e6320696e20636f6d6d6f646f206d692c20657520736f64616c6573206e756e632e20416c697175616d206c7563747573206c6f72656d2065742074696e636964756e74206c6163696e69612e20447569732076656c20697073756d206e69736c2e205072616573656e7420636f6e76616c6c697320656c6974206c6967756c612c206e656320706f72746120657374206d6178696d75732061632e204e756c6c61207072657469756d206c696265726f206567657420616e746520756c6c616d636f72706572206d61747469732e204e756c6c616d20766f6c75747061742c2074656c6c757320736564207363656c65726973717565206566666963697475722c206e69736c2061756775652070686172657472612066656c69732c2076656c2067726176696461206d61676e612075726e6120736564207175616d2e2044756973206964207072657469756d206475692e20496e74656765722072686f6e637573206d6174746973206a7573746f20612068656e6472657269742e20467573636520646f6c6f72206d61676e612c20706f72747469746f7220616320707572757320736f64616c65732c20657569736d6f6420766573746962756c756d20746f72746f722e20416c697175616d2070686172657472612065726174206a7573746f2c20696e20756c6c616d636f72706572207175616d2e"
                ).to_vec(),
            })
        );

        // Catch-All
        assert_eq!(vaa, val);
    }

    #[test]
    fn test_guardian_set_upgrade() {
        let vaa = hex_literal::hex!("010000000001007ac31b282c2aeeeb37f3385ee0de5f8e421d30b9e5ae8ba3d4375c1c77a86e77159bb697d9c456d6f8c02d22a94b1279b65b0d6a9957e7d3857423845ac758e300610ac1d2000000030001000000000000000000000000000000000000000000000000000000000000000400000000000005390000000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570746aa5ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cdeb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768eaf45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!(
                    "007ac31b282c2aeeeb37f3385ee0de5f8e421d30b9e5ae8ba3d4375c1c77a86e77159bb697d9c456d6f8c02d22a94b1279b65b0d6a9957e7d3857423845ac758e300"
                )
            ],
            timestamp:          1628094930,
            nonce:              3,
            emitter_chain:      1.into(),
            emitter_address:    hex_literal::hex!(
                "0000000000000000000000000000000000000000000000000000000000000004"
            ),
            sequence:           1337,
            consistency_level:  0,
            payload:            hex_literal::hex!("
                00000000000000000000000000000000000000000000000000000000436f7265020000000000011358cc3ae5c097b213ce3c81979e1b9f9570746aa5
                ff6cb952589bde862c25ef4392132fb9d4a42157114de8460193bdf3a2fcf81f86a09765f4762fd1107a0086b32d7a0977926a205131d8731d39cbeb
                8c82b2fd82faed2711d59af0f2499d16e726f6b211b39756c042441be6d8650b69b54ebe715e234354ce5b4d348fb74b958e8966e2ec3dbd4958a7cd
                eb5f7389fa26941519f0863349c223b73a6ddee774a3bf913953d695260d88bc1aa25a4eee363ef0000ac0076727b35fbea2dac28fee5ccb0fea768e
                af45ced136b9d9e24903464ae889f5c8a723fc14f93124b7c738843cbb89e864c862c38cddcccf95d2cc37a4dc036a8d232b48f62cdd4731412f4890
                da798f6896a3331f64b48c12d1d57fd9cbe7081171aa1be1d36cafe3867910f99c09e347899c19c38192b6e7387ccd768277c17dab1b7a5027c0b3cf
                178e21ad2e77ae06711549cfbb1f9c7a9d8096e85e1487f35515d02a92753504a8d75471b9f49edb6fbebc898f403e4773e95feb15e80c9a99c8348d
            ").to_vec(),
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            core::Action::from_vaa(&vaa, Chain::Any).unwrap(),
            core::Action::GuardianSetChange(core::GuardianSetChange {
                header:                 GovHeader {
                    module: core::MODULE,
                    action: 0x02,
                    target: Chain::Any,
                },
                new_guardian_set_index: 1,
                new_guardian_set:       vec![
                    hex_literal::hex!("58cc3ae5c097b213ce3c81979e1b9f9570746aa5"),
                    hex_literal::hex!("ff6cb952589bde862c25ef4392132fb9d4a42157"),
                    hex_literal::hex!("114de8460193bdf3a2fcf81f86a09765f4762fd1"),
                    hex_literal::hex!("107a0086b32d7a0977926a205131d8731d39cbeb"),
                    hex_literal::hex!("8c82b2fd82faed2711d59af0f2499d16e726f6b2"),
                    hex_literal::hex!("11b39756c042441be6d8650b69b54ebe715e2343"),
                    hex_literal::hex!("54ce5b4d348fb74b958e8966e2ec3dbd4958a7cd"),
                    hex_literal::hex!("eb5f7389fa26941519f0863349c223b73a6ddee7"),
                    hex_literal::hex!("74a3bf913953d695260d88bc1aa25a4eee363ef0"),
                    hex_literal::hex!("000ac0076727b35fbea2dac28fee5ccb0fea768e"),
                    hex_literal::hex!("af45ced136b9d9e24903464ae889f5c8a723fc14"),
                    hex_literal::hex!("f93124b7c738843cbb89e864c862c38cddcccf95"),
                    hex_literal::hex!("d2cc37a4dc036a8d232b48f62cdd4731412f4890"),
                    hex_literal::hex!("da798f6896a3331f64b48c12d1d57fd9cbe70811"),
                    hex_literal::hex!("71aa1be1d36cafe3867910f99c09e347899c19c3"),
                    hex_literal::hex!("8192b6e7387ccd768277c17dab1b7a5027c0b3cf"),
                    hex_literal::hex!("178e21ad2e77ae06711549cfbb1f9c7a9d8096e8"),
                    hex_literal::hex!("5e1487f35515d02a92753504a8d75471b9f49edb"),
                    hex_literal::hex!("6fbebc898f403e4773e95feb15e80c9a99c8348d")
                ],
            })
        );

        // Catch-All
        assert_eq!(vaa.payload, val.payload);
    }

    #[test]
    fn test_nft_bridge_transfer() {
        let vaa = hex_literal::hex!("010000000000000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000277bb0b0001000000000000000000000000000000000000000000000000000000000000000400010000000000000000000000000000000000000000000000000000000000464f4f0000000000000000000000000000000000000000000000000000000000424152000000000000000000000000000000000000000000000000000000000000000a0a676f6f676c652e636f6d0000000000000000000000000000000000000000000000000000000000000004000a");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![],
            timestamp:          1,
            nonce:              1,
            emitter_chain:      Chain::Solana,
            emitter_address:    hex_literal::hex!(
                "0000000000000000000000000000000000000000000000000000000000000004"
            ),
            sequence:           41401099,
            consistency_level:  0,
            payload:            vec![
                1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 4, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 70, 79, 79, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 66, 65, 82, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10, 10, 103, 111, 111,
                103, 108, 101, 46, 99, 111, 109, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 10,
            ],
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            nft::Action::from_vaa(&vaa, Chain::Any).unwrap(),
            nft::Action::Transfer(nft::Transfer {
                token_id:    10.into(),
                symbol:      "FOO".to_string(),
                name:        "BAR".to_string(),
                uri:         "google.com".to_string(),
                to:          hex_literal::hex!(
                    "0000000000000000000000000000000000000000000000000000000000000004"
                ),
                to_chain:    Chain::Fantom,
                nft_chain:   Chain::Solana,
                nft_address: hex_literal::hex!(
                    "0000000000000000000000000000000000000000000000000000000000000004"
                ),
            })
        );

        // Catch-All
        assert_eq!(vaa, val);
    }

    #[test]
    fn token_bridge_attestation() {
        let vaa = hex_literal::hex!("010000000001006cd3cdd701bbd878eb403f6505b5b797544eb9c486dadf79f0c445e9b8fa5cd474de1683e3a80f7e22dbfacd53b0ddc7b040ff6f974aafe7a6571c9355b8129b00000000007ce2ea3f000195f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491a43a1c0020f88a3e2002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200021200000000000000000000000000000000000000000000000000000000574554480000000000000000000000000000000000000057726170706564206574686572");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!("006cd3cdd701bbd878eb403f6505b5b797544eb9c486dadf79f0c445e9b8fa5cd474de1683e3a80f7e22dbfacd53b0ddc7b040ff6f974aafe7a6571c9355b8129b00"),
            ],
            timestamp:          0,
            nonce:              2095245887,
            emitter_chain:      Chain::Solana,
            emitter_address:    hex_literal::hex!(
                "95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491"
            ),
            sequence:           11833801757748136510,
            consistency_level:  32,
            payload:            hex_literal::hex!("02000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc200021200000000000000000000000000000000000000000000000000000000574554480000000000000000000000000000000000000057726170706564206574686572").to_vec(),
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            token::Action::from_vaa(&vaa, Chain::Any).unwrap(),
            token::Action::AssetMeta(token::AssetMeta {
                symbol:        "WETH".to_string(),
                name:          "Wrapped ether".to_string(),
                decimals:      18,
                token_address: hex_literal::hex!(
                    "000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
                ),
                token_chain:   Chain::Ethereum,
            })
        );

        // Catch-All
        assert_eq!(vaa, val);
    }

    #[test]
    fn token_bridge_transfer() {
        let vaa = hex_literal::hex!("010000000001007d204ad9447c4dfd6be62406e7f5a05eec96300da4048e70ff530cfb52aec44807e98194990710ff166eb1b2eac942d38bc1cd6018f93662a6578d985e87c8d0016221346b0000b8bd0001c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f0000000000000003200100000000000000000000000000000000000000000000000000000002540be400165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3010001000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d00020000000000000000000000000000000000000000000000000000000000000000");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!("007d204ad9447c4dfd6be62406e7f5a05eec96300da4048e70ff530cfb52aec44807e98194990710ff166eb1b2eac942d38bc1cd6018f93662a6578d985e87c8d001"),
            ],
            timestamp:          1646343275,
            nonce:              47293,
            emitter_chain:      Chain::Solana,
            emitter_address:    hex_literal::hex!("c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f"),
            sequence:           3,
            consistency_level:  32,
            payload:            hex_literal::hex!("0100000000000000000000000000000000000000000000000000000002540be400165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3010001000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d00020000000000000000000000000000000000000000000000000000000000000000").to_vec(),
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            token::Action::from_vaa(&vaa, Chain::Any).unwrap(),
            token::Action::Transfer(token::Transfer {
                amount:        10000000000usize.into(),
                token_address: hex_literal::hex!(
                    "165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa301"
                ),
                token_chain:   Chain::Solana,
                to:            hex_literal::hex!(
                    "000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d"
                ),
                to_chain:      Chain::Ethereum,
                fee:           0.into(),
            })
        );

        // Catch-All
        println!("{}", hex::encode(&vaa.payload));
        assert_eq!(vaa, val);
    }

    #[test]
    fn token_bridge_upgrade() {
        let vaa = hex_literal::hex!("01000000000100e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe7009000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004c5d05a00000000000000000000000000000000000000000000546f6b656e42726964676502000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2");
        let vaa = VAA::from_bytes(vaa).unwrap();
        let val = VAA {
            version:            1,
            guardian_set_index: 0,
            signatures:         vec![
                hex_literal::hex!("00e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe700900"),
            ],
            timestamp:          1,
            nonce:              1,
            emitter_chain:      Chain::Solana,
            emitter_address:    hex_literal::hex!("0000000000000000000000000000000000000000000000000000000000000004"),
            sequence:           80072794,
            consistency_level:  0,
            payload:            hex_literal::hex!("000000000000000000000000000000000000000000546f6b656e42726964676502000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2").to_vec(),
        };

        assert_eq!(vaa.version, val.version);
        assert_eq!(vaa.guardian_set_index, val.guardian_set_index);
        assert_eq!(vaa.signatures, val.signatures);
        assert_eq!(vaa.timestamp, val.timestamp);
        assert_eq!(vaa.nonce, val.nonce);
        assert_eq!(vaa.emitter_chain, val.emitter_chain);
        assert_eq!(vaa.emitter_address, val.emitter_address);
        assert_eq!(vaa.sequence, val.sequence);
        assert_eq!(vaa.consistency_level, val.consistency_level);
        assert_eq!(
            token::Action::from_vaa(&vaa, Chain::Fantom).unwrap(),
            token::Action::ContractUpgrade(token::ContractUpgrade {
                new_contract: hex_literal::hex!(
                    "0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2"
                ),
            })
        );

        // Catch-All
        println!("{}", hex::encode(&vaa.payload));
        assert_eq!(vaa, val);
    }

    // Bench VAA::from_byte nom based parser.
    #[bench]
    fn bench_vaa_parse(b: &mut test::Bencher) {
        let vaa = hex_literal::hex!("01000000000100e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe7009000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004c5d05a00000000000000000000000000000000000000000000546f6b656e42726964676502000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2");
        b.iter(|| {
            let _ = VAA::from_bytes(vaa);
        });
    }

    // Bench original `legacy_deserialize`.
    #[bench]
    fn bench_legacy_vaa_parse(b: &mut test::Bencher) {
        let vaa = hex_literal::hex!("01000000000100e3db309303b712a562e6aa2adc68bc10ff22328ab31ddb6a83706943a9da97bf11ba6e3b96395515868786898dc19ecd737d197b0d1a1f3f3c6aead5c1fe7009000000000100000001000100000000000000000000000000000000000000000000000000000000000000040000000004c5d05a00000000000000000000000000000000000000000000546f6b656e42726964676502000a0000000000000000000000000046da7a0320dd999438b4435dac82bf1dac13d2");
        b.iter(|| {
            let _ = legacy_deserialize(&vaa);
        });
    }
}
