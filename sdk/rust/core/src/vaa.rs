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

use std::io::{self, Write};

use anyhow::Context;
use serde::{Deserialize, Serialize};
use sha3::Digest as Sha3Digest;

use crate::{Address, Chain, GuardianAddress};

/// Signatures are typical ECDSA signatures prefixed with a Guardian position. These have the
/// following byte layout:
/// ```markdown
/// 0  .. 64: Signature   (ECDSA)
/// 64 .. 65: Recovery ID (ECDSA)
/// ```
#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
#[cfg_attr(feature = "schemars", derive(schemars::JsonSchema))]
pub struct Signature {
    pub index: u8,
    #[serde(with = "crate::serde_array")]
    #[schemars(with = "schemars_array::Array<u8, 65>")]
    pub signature: [u8; 65],
}

impl Default for Signature {
    fn default() -> Self {
        Self {
            index: 0,
            signature: [0; 65],
        }
    }
}

// This is a workaround for the fact that trait impls for arrays only go up to 32 elements.
#[cfg(feature = "schemars")]
mod schemars_array {
    use schemars::{
        gen::SchemaGenerator,
        schema::{ArrayValidation, InstanceType, Schema, SchemaObject},
        JsonSchema,
    };

    pub struct Array<T, const N: usize>(pub [T; N]);

    impl<T: JsonSchema, const N: usize> JsonSchema for Array<T, N> {
        fn is_referenceable() -> bool {
            false
        }

        fn schema_name() -> String {
            format!("Array_size_{N}_of_{}", T::schema_name())
        }

        fn json_schema(gen: &mut SchemaGenerator) -> Schema {
            SchemaObject {
                instance_type: Some(InstanceType::Array.into()),
                array: Some(Box::new(ArrayValidation {
                    items: Some(gen.subschema_for::<T>().into()),
                    max_items: Some(N as u32),
                    min_items: Some(N as u32),
                    ..Default::default()
                })),
                ..Default::default()
            }
            .into()
        }
    }
}

/// The core VAA itself. This structure is what is received by a contract on the receiving side of
/// a wormhole message passing flow.  The generic parameter `P` represents the user-defined payload
/// for the VAA.
#[derive(Serialize, Deserialize, Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Vaa<P> {
    // Implementation note: it would be nice if we could use `#[serde(flatten)]` and directly embed the
    // `Header` and `Body` structs. Unfortunately using flatten causes serde to serialize/deserialize
    // the struct as a map, which requires the underlying data format to encode field names on the
    // wire, which the wormhole data format does not do.  So instead we have to duplicate the fields
    // and provide a conversion function to/from `Vaa` to `(Header, Body)`.
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: Chain,
    pub emitter_address: Address,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: P,
}

/// The header for a VAA.
#[derive(Serialize, Deserialize, Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Header {
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,
}

/// The body for a VAA.
#[derive(Serialize, Deserialize, Debug, Default, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Body<P> {
    /// Seconds since UNIX epoch.
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: Chain,
    pub emitter_address: Address,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: P,
}

/// Digest data for the Body.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Digest {
    /// Guardians don't hash the VAA body directly, instead they hash the VAA and sign the hash. The
    /// purpose of this is it means when submitting a VAA on-chain we only have to submit the hash
    /// which reduces gas costs.
    pub hash: [u8; 32],

    /// The secp256k_hash is the hash of the hash of the VAA. The reason we provide this is because
    /// of how secp256k works internally. It hashes its payload before signing. This means that
    /// when verifying secp256k signatures, we're actually checking if a guardian has signed the
    /// hash of the hash of the VAA. Functions such as `ecrecover` expect the secp256k hash rather
    /// than the original payload.
    pub secp256k_hash: [u8; 32],
}

/// Calculates and returns the digest for `body` to be used in VAA operations.
///
/// A VAA is distinguished by the unique 256bit Keccak256 hash of its body. This hash is
/// utilised in all Wormhole components for identifying unique VAA's, including the bridge,
/// modules, and core guardian software. The `Digest` is documented with reasoning for
/// each field.
///
/// NOTE: This function uses a library to do Keccak256 hashing, but on-chain this may not be
/// efficient. If efficiency is needed, consider calling `body()` instead and hashing the
/// result using on-chain primitives.
pub fn digest(body: &[u8]) -> io::Result<Digest> {
    // The `body` of the VAA is hashed to produce a `digest` of the VAA.
    let hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write_all(body)?;
        h.finalize().into()
    };

    // Hash `hash` again to get the secp256k internal hash, see `Digest` for more details.
    let secp256k_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write_all(&hash)?;
        h.finalize().into()
    };

    Ok(Digest {
        hash,
        secp256k_hash,
    })
}

impl<P> Vaa<P> {
    /// Check if the VAA is a Governance VAA.
    pub fn is_governance(&self) -> bool {
        self.emitter_address == crate::GOVERNANCE_EMITTER && self.emitter_chain == Chain::Solana
    }
}

impl<P> From<Vaa<P>> for (Header, Body<P>) {
    fn from(v: Vaa<P>) -> Self {
        (
            Header {
                version: v.version,
                guardian_set_index: v.guardian_set_index,
                signatures: v.signatures,
            },
            Body {
                timestamp: v.timestamp,
                nonce: v.nonce,
                emitter_chain: v.emitter_chain,
                emitter_address: v.emitter_address,
                sequence: v.sequence,
                consistency_level: v.consistency_level,
                payload: v.payload,
            },
        )
    }
}

impl<P> From<(Header, Body<P>)> for Vaa<P> {
    fn from((hdr, body): (Header, Body<P>)) -> Self {
        Vaa {
            version: hdr.version,
            guardian_set_index: hdr.guardian_set_index,
            signatures: hdr.signatures,
            timestamp: body.timestamp,
            nonce: body.nonce,
            emitter_chain: body.emitter_chain,
            emitter_address: body.emitter_address,
            sequence: body.sequence,
            consistency_level: body.consistency_level,
            payload: body.payload,
        }
    }
}

impl Header {
    pub fn verify(&self, _body: &[u8], _addrs: &[GuardianAddress]) -> anyhow::Result<Digest> {
        todo!("VAA body verification")
    }
}

impl<P> Body<P> {
    /// Replace the payload of the body.  Useful when parsing the payload needs to be delayed.
    pub fn with_payload<U>(self, p: U) -> Body<U> {
        Body {
            timestamp: self.timestamp,
            nonce: self.nonce,
            emitter_chain: self.emitter_chain,
            emitter_address: self.emitter_address,
            sequence: self.sequence,
            consistency_level: self.consistency_level,
            payload: p,
        }
    }
}

impl<P: Serialize> Body<P> {
    /// Body Digest Components.
    ///
    /// A VAA is distinguished by the unique 256bit Keccak256 hash of its body. This hash is
    /// utilised in all Wormhole components for identifying unique VAA's, including the bridge,
    /// modules, and core guardian software. The `Digest` is documented with reasoning for
    /// each field.
    ///
    /// NOTE: This function uses a library to do Keccak256 hashing, but on-chain this may not be
    /// efficient. If efficiency is needed, consider calling `serde_wormhole::to_writer` instead
    /// and hashing the result using on-chain primitives.
    #[inline]
    pub fn digest(&self) -> anyhow::Result<Digest> {
        // The `body` of the VAA is hashed to produce a `digest` of the VAA.
        let hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            serde_wormhole::to_writer(&mut h, self).context("failed to serialize body")?;
            h.finalize().into()
        };

        // Hash `hash` again to get the secp256k internal hash, see `Digest` for detail.
        let secp256k_hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write_all(&hash)
                .context("failed to compute second hash")?;
            h.finalize().into()
        };

        Ok(Digest {
            hash,
            secp256k_hash,
        })
    }
}

#[cfg(test)]
mod test {
    use serde_wormhole::RawMessage;

    use crate::{
        token::{Action, GovernancePacket},
        GOVERNANCE_EMITTER,
    };

    use super::*;

    #[test]
    fn arbitrary_payload() {
        let buf = [
            0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0xe2, 0x9d, 0x3a, 0xd1, 0x80, 0xb1, 0x53,
            0xd6, 0x8c, 0x3f, 0x44, 0x5d, 0x75, 0xea, 0xa6, 0x2f, 0xcc, 0x99, 0x69, 0x09, 0x45,
            0xba, 0xaf, 0x4a, 0xd0, 0x46, 0x3e, 0x9c, 0xe4, 0x4f, 0x27, 0xf7, 0x5d, 0xa3, 0xd4,
            0x9f, 0x79, 0x72, 0x29, 0x20, 0xaa, 0xc8, 0x1b, 0xa2, 0xbe, 0x80, 0xf6, 0x88, 0x89,
            0x5f, 0x17, 0x49, 0x42, 0xfe, 0xdc, 0x40, 0x3b, 0xc4, 0xe5, 0xce, 0x35, 0x55, 0xb7,
            0x7b, 0x00, 0x62, 0xb9, 0xf7, 0x91, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf1, 0x9a, 0x2a, 0x01,
            0xb7, 0x05, 0x19, 0xf6, 0x7a, 0xdb, 0x30, 0x9a, 0x99, 0x4e, 0xc8, 0xc6, 0x9a, 0x96,
            0x7e, 0x8b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x46, 0x72, 0x6f,
            0x6d, 0x3a, 0x20, 0x65, 0x76, 0x6d, 0x30, 0x5c, 0x6e, 0x4d, 0x73, 0x67, 0x3a, 0x20,
            0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57, 0x6f, 0x72, 0x6c, 0x64, 0x21,
        ];

        let vaa = Vaa {
            version: 1,
            guardian_set_index: 0,
            signatures: vec![Signature {
                index: 0,
                signature: [
                    0xe2, 0x9d, 0x3a, 0xd1, 0x80, 0xb1, 0x53, 0xd6, 0x8c, 0x3f, 0x44, 0x5d, 0x75,
                    0xea, 0xa6, 0x2f, 0xcc, 0x99, 0x69, 0x09, 0x45, 0xba, 0xaf, 0x4a, 0xd0, 0x46,
                    0x3e, 0x9c, 0xe4, 0x4f, 0x27, 0xf7, 0x5d, 0xa3, 0xd4, 0x9f, 0x79, 0x72, 0x29,
                    0x20, 0xaa, 0xc8, 0x1b, 0xa2, 0xbe, 0x80, 0xf6, 0x88, 0x89, 0x5f, 0x17, 0x49,
                    0x42, 0xfe, 0xdc, 0x40, 0x3b, 0xc4, 0xe5, 0xce, 0x35, 0x55, 0xb7, 0x7b, 0x00,
                ],
            }],
            timestamp: 1_656_354_705,
            nonce: 0,
            emitter_chain: Chain::Ethereum,
            emitter_address: Address([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf1, 0x9a,
                0x2a, 0x01, 0xb7, 0x05, 0x19, 0xf6, 0x7a, 0xdb, 0x30, 0x9a, 0x99, 0x4e, 0xc8, 0xc6,
                0x9a, 0x96, 0x7e, 0x8b,
            ]),
            sequence: 0,
            consistency_level: 1,
            payload: RawMessage::new(&buf[123..]),
        };

        assert_eq!(vaa, serde_wormhole::from_slice(&buf).unwrap());
        assert_eq!(&buf[..], &serde_wormhole::to_vec(&vaa).unwrap());
    }

    #[test]
    fn digest_from_raw_parts() {
        let body = Body {
            timestamp: 1_656_354_705,
            nonce: 0,
            emitter_chain: Chain::Ethereum,
            emitter_address: Address([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf1, 0x9a,
                0x2a, 0x01, 0xb7, 0x05, 0x19, 0xf6, 0x7a, 0xdb, 0x30, 0x9a, 0x99, 0x4e, 0xc8, 0xc6,
                0x9a, 0x96, 0x7e, 0x8b,
            ]),
            sequence: 0,
            consistency_level: 1,
            payload: "From: evm0\\nMsg: Hello World!",
        };

        let d1 = body.digest().unwrap();

        let data = serde_wormhole::to_vec(&body).unwrap();
        let d2 = digest(&data).unwrap();

        assert_eq!(d1, d2);

        let partial = serde_wormhole::from_slice::<Body<&RawMessage>>(&data).unwrap();
        let d3 = partial.digest().unwrap();

        assert_eq!(d1, d3);
    }

    #[test]
    fn stable_digest() {
        let data = [
            0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
            0x00, 0x00, 0x00, 0x00, 0x03, 0xb4, 0x56, 0xb8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x72, 0x69, 0x64, 0x67, 0x65, 0x01,
            0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x02, 0x90, 0xfb, 0x16, 0x72, 0x08, 0xaf, 0x45, 0x5b, 0xb1, 0x37, 0x78,
            0x01, 0x63, 0xb7, 0xb7, 0xa9, 0xa1, 0x0c, 0x16,
        ];

        let expected_digest = [
            0x05, 0xd1, 0xfc, 0xc5, 0x31, 0x74, 0x6c, 0x7e, 0xfd, 0x7f, 0xee, 0xa2, 0x0a, 0x81,
            0xd2, 0x79, 0x9f, 0x77, 0x7f, 0x30, 0x2b, 0x8a, 0x6a, 0x64, 0x24, 0xb8, 0x12, 0x09,
            0xdc, 0x3f, 0x51, 0x1f,
        ];

        assert_eq!(expected_digest, digest(&data).unwrap().secp256k_hash);

        let expected_body = Body {
            timestamp: 1,
            nonce: 1,
            emitter_chain: Chain::Solana,
            emitter_address: GOVERNANCE_EMITTER,
            sequence: 62150328,
            consistency_level: 0,
            payload: GovernancePacket {
                chain: Chain::Any,
                action: Action::RegisterChain {
                    chain: Chain::Ethereum,
                    emitter_address: Address([
                        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                        0x02, 0x90, 0xfb, 0x16, 0x72, 0x08, 0xaf, 0x45, 0x5b, 0xb1, 0x37, 0x78,
                        0x01, 0x63, 0xb7, 0xb7, 0xa9, 0xa1, 0x0c, 0x16,
                    ]),
                },
            },
        };

        let body = serde_wormhole::from_slice(&data).unwrap();
        assert_eq!(expected_body, body);
        assert_eq!(expected_digest, body.digest().unwrap().secp256k_hash);

        // Deferred parsing of the payload should still produce the same digest.
        let body = serde_wormhole::from_slice::<Body<&RawMessage>>(&data).unwrap();
        assert_eq!(&data[51..], body.payload.get());
        assert_eq!(expected_digest, body.digest().unwrap().secp256k_hash);
    }
}
