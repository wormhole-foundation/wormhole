//! Serialize and deserialize rust values from the VAA payload wire format.
//!
//! As of this writing (June, 2022) there is no proper specification for the VAA payload wire
//! format so this implementation has mostly been reverse engineered from the existing messages.
//! While the rest of this document talks about how various types are represented on the wire this
//! should be seen as an explanation of how things are implemented *in this crate* and not as
//! official documentation. In cases where the serialization of a payload produced by this crate
//! differs from the one use by the wormhole contracts, the serialization used by the actual
//! contract is considered the canonical serialization.
//!
//! Unless you want to interact with existing wormhole VAA payloads, this crate is probably not what
//! you are looking for. If you are simply using the wormhole bridge to send your own payloads then
//! using a schema with auto-generated code (like protobufs or flatbuffers) is probably a better
//! choice.
//!
//! ## Wire format
//!
//! The VAA payload wire format is not a self-describing format (unlike json and toml). Therefore it
//! is necessary to know the type that needs to be produced before deserializing a byte stream.
//!
//! The wire format currently supports the following primitive types:
//!
//! ### `bool`
//!
//! Encoded as a single byte where a value of 0 indicates false and 1 indicates true. All other
//! values are invalid.
//!
//! ### Integers
//!
//! `i8`, `i16`, `i32`, `i64`, `i128`, `u8`, `u16`, `u32`, `u64`, and `u128` are all supported and
//! encoded as full-width big-endian integers, i.e., `i16` is 2 bytes, `u64` is 8 bytes, etc.
//!
//! ### `char`
//!
//! Encoded as a big-endian `u32`, with the additional restriction that it must be a valid [`Unicode
//! Scalar Value`](https://www.unicode.org/glossary/#unicode_scalar_value).
//!
//! ### Sequences
//!
//! Variable length heterogeneous sequences are encoded as a single byte length followed by the
//! concatenation of the serialized form of each element in the sequence. Note that this means that
//! sequences cannot have more than 255 elements. Additionally, during serialization the length must
//! be known ahead of time.
//!
//! ### Byte arrays - `&[u8]`, `Vec<u8>`, and `Cow<'a, [u8]>`
//!
//! Byte arrays are treated as a subset of variable-length sequences and are encoded as a single
//! byte length followed by that many bytes of data. Again, since the length of the byte array has
//! to fit in a single byte it cannot be longer than 255 bytes.
//!
//! ### `&str`, `String`
//!
//! String types are encoded the same way as `&[u8]`, with the additional restriction that the byte
//! array must be valid UTF-8.
//!
//! ### Tuples
//!
//! Tuples are heterogenous sequences where the length is fixed and known ahead of time. In this
//! case the length is not encoded on the wire and the serialization of each element in the tuple is
//! concatenated to produce the final value.
//!
//! ### `Option<T>`
//!
//! The wire format does not support optional values. Options are always deserialized as `Some(T)`
//! while trying to serialize an `Option::None` will result in an error.
//!
//! ### Structs
//!
//! Structs are represented the same way as tuples and the wire format for a struct is identical to
//! the wire format for a tuple with the same fields in the same order. The only exception is unit
//! structs (structs with no fields), which are not represented in the wire format at all.
//!
//! ### `[T; N]`
//!
//! Arrays are treated as tuples with homogenous fields and have the same wire format.
//!
//! ### Enums
//!
//! Enums are encoded as a single byte identifying the variant followed by the serialization of the
//! variant.
//!
//! * Unit variants - No additional data is encoded.
//! * Newtype variants - Encoded using the serialization of the inner type.
//! * Tuple variants - Encoded as a regular tuple.
//! * Struct variants - Encoded as a regular struct.
//!
//! Since the enum variant is encoded as a single byte rather than the name of the variant itself,
//! it is necessary to use `#[serde(rename = "<value>")]` on each enum variant to ensure
//! that they can be serialized and deserialized properly.
//!
//! #### Examples
//!
//! ```
//! use std::borrow::Cow;
//!
//! use serde::{Deserialize, Serialize};
//!
//! #[derive(Serialize, Deserialize)]
//! enum TestEnum<'a> {
//!     #[serde(rename = "19")]
//!     Unit,
//!     #[serde(rename = "235")]
//!     NewType(u64),
//!     #[serde(rename = "179")]
//!     Tuple(u32, u64, Vec<u16>),
//!     #[serde(rename = "97")]
//!     Struct {
//!         #[serde(borrow, with = "serde_bytes")]
//!         data: Cow<'a, [u8]>,
//!         footer: u32,
//!     },
//! }
//!
//! assert!(matches!(serde_wormhole::from_slice(&[19]).unwrap(), TestEnum::Unit));
//! ```
//!
//! ### Map types
//!
//! Map types are encoded as a sequence of `(key, value)` tuples. The encoding for a `Vec<(K, V)>`
//! is identical to that of a `BTreeMap<K, V>`. During serialiazation, the number of elements in the
//! map must be known ahead of time. Like other sequences, the maximum number of elements in the map
//! is 255.

use std::io::{Read, Write};

use serde::{de::DeserializeOwned, Deserialize, Serialize};

mod de;
mod error;
mod ser;

pub use error::Error;

/// Deserialize an instance of type `T` from the provided reader.
pub fn from_reader<R: Read, T: DeserializeOwned>(mut r: R) -> Result<T, Error> {
    // We can do something smarter here by making the deserializer generic over the reader (see
    // serde_json::Deserializer) but for now this is probably good enough.
    let mut buf = Vec::with_capacity(128);
    r.read_to_end(&mut buf)?;

    from_slice(&buf)
}

/// Like `from_reader` but also returns any trailing data in the input buffer after
/// deserialization.
pub fn from_reader_with_payload<R: Read, T: DeserializeOwned>(
    mut r: R,
) -> Result<(T, Vec<u8>), Error> {
    // We can do something smarter here by making the deserializer generic over the reader (see
    // serde_json::Deserializer) but for now this is probably good enough.
    let mut buf = Vec::with_capacity(128);
    r.read_to_end(&mut buf)?;

    from_slice_with_payload(&buf).map(|(v, p)| (v, p.to_vec()))
}

/// Deserialize an instance of type `T` from a byte slice.
pub fn from_slice<'a, T: Deserialize<'a>>(buf: &'a [u8]) -> Result<T, Error> {
    let mut deserializer = de::Deserializer::new(buf);

    let v = T::deserialize(&mut deserializer)?;

    if deserializer.end().is_empty() {
        Ok(v)
    } else {
        Err(Error::TrailingData)
    }
}

/// Like `from_slice` but also returns any trailing data in the input buffer after deserialization.
pub fn from_slice_with_payload<'a, T: Deserialize<'a>>(
    buf: &'a [u8],
) -> Result<(T, &'a [u8]), Error> {
    let mut deserializer = de::Deserializer::new(buf);

    T::deserialize(&mut deserializer).map(|v| (v, deserializer.end()))
}

/// Serialize `T` into a byte vector.
pub fn to_vec<T: ?Sized + Serialize>(val: &T) -> Result<Vec<u8>, Error> {
    let mut buf = Vec::with_capacity(128);

    to_writer(&mut buf, val)?;
    Ok(buf)
}

/// Serialize `T` into the provided writer.
pub fn to_writer<W: Write, T: ?Sized + Serialize>(w: W, val: &T) -> Result<(), Error> {
    let mut serializer = ser::Serializer::new(w);
    val.serialize(&mut serializer)
}

#[cfg(test)]
mod tests {
    use core::panic;
    use std::{borrow::Cow, collections::BTreeMap};

    use super::*;

    use serde::{Deserialize, Serialize};
    use serde_repr::{Deserialize_repr, Serialize_repr};

    mod serde_array {
        use std::{fmt, mem::MaybeUninit};

        use serde::{
            de::{Error, SeqAccess, Visitor},
            ser::SerializeTuple,
            Deserializer, Serializer,
        };

        pub fn serialize<const N: usize, S>(
            value: &[u8; N],
            serializer: S,
        ) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            let mut seq = serializer.serialize_tuple(N)?;
            for v in value {
                seq.serialize_element(v)?;
            }

            seq.end()
        }

        struct ArrayVisitor<const N: usize>;
        impl<'de, const N: usize> Visitor<'de> for ArrayVisitor<N> {
            type Value = [u8; N];

            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                write!(formatter, "an array of length {}", N)
            }

            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                // TODO: Replace with `MaybeUninit::uninit_array()` once that's stabilized.
                let mut buf = MaybeUninit::<[u8; N]>::uninit();
                let ptr = buf.as_mut_ptr() as *mut u8;
                let mut pos = 0;

                while pos < N {
                    let v = seq
                        .next_element()
                        .and_then(|v| v.ok_or_else(|| Error::invalid_length(pos, &self)))?;

                    // Safety: The resulting pointer is within the bounds of the allocation because
                    // we know that `pos < N`.
                    unsafe { ptr.add(pos).write(v) };

                    pos += 1;
                }

                if pos == N {
                    // Safety: We've initialized all the bytes in `buf`.
                    Ok(unsafe { buf.assume_init() })
                } else {
                    Err(Error::invalid_length(pos, &self))
                }
            }
        }

        pub fn deserialize<'de, const N: usize, D>(deserializer: D) -> Result<[u8; N], D::Error>
        where
            D: Deserializer<'de>,
        {
            deserializer.deserialize_tuple(N, ArrayVisitor)
        }
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct Header {
        version: u8,
        guardian_set_index: u32,
    }

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
    struct Signature {
        index: u8,
        #[serde(with = "serde_array")]
        signature: [u8; 65],
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct Vaa<'s> {
        header: Header,
        #[serde(borrow)]
        signatures: Cow<'s, [Signature]>,
        timestamp: u32, // Seconds since UNIX epoch
        nonce: u32,
        emitter_chain: u16,
        emitter_address: [u8; 32],
        sequence: u64,
        consistency_level: u8,
        map: BTreeMap<u32, u32>,
        payload: GovernancePacket,
    }

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
    struct GuardianAddress<'a> {
        #[serde(borrow, with = "serde_bytes")]
        bytes: Cow<'a, [u8]>,
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct GuardianSetInfo<'a> {
        #[serde(borrow)]
        addresses: Cow<'a, [GuardianAddress<'a>]>,
        expiration_time: u64,
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct ContractUpgrade {
        new_contract: u64,
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct GuardianSetUpgrade<'a> {
        new_guardian_set_index: u32,
        #[serde(borrow)]
        new_guardian_set: GuardianSetInfo<'a>,
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct SetFee {
        high: u128,
        low: u128,
    }

    #[derive(Debug, Serialize_repr, Deserialize_repr, PartialEq, Eq)]
    #[repr(u8)]
    enum Action {
        ContractUpgrade = 1,
        GuardianSetUpgrade = 2,
        SetFee = 3,
    }

    #[derive(Debug, Serialize_repr, Deserialize_repr, PartialEq, Eq)]
    #[repr(u16)]
    enum Chain {
        Unset = 0,
        Solana = 1,
        Ethereum = 2,
        Terra = 3,
    }

    #[derive(Debug, Serialize, Deserialize, PartialEq, Eq)]
    struct GovernancePacket {
        module: [u8; 32],
        action: Action,
        chain: Chain,
    }

    #[test]
    fn end_to_end() {
        let vaa = Vaa {
            header: Header {
                version: 3,
                guardian_set_index: 0x97a5_6966,
            },
            signatures: Cow::Borrowed(&[
                Signature {
                    index: 0x13,
                    signature: [
                        0x23, 0x35, 0xf3, 0xc2, 0x2c, 0xd2, 0x43, 0xf4, 0xcd, 0xe4, 0x7a, 0xa9,
                        0xdd, 0x99, 0x35, 0xbc, 0x20, 0x8f, 0x9c, 0x2d, 0x2e, 0xa4, 0x8e, 0xe0,
                        0x85, 0x89, 0x33, 0x65, 0x0b, 0x8c, 0x6c, 0x14, 0xd9, 0x6b, 0x41, 0xe8,
                        0x4b, 0xc7, 0xef, 0xae, 0x75, 0x3d, 0x9f, 0x1a, 0x36, 0x4c, 0x09, 0x62,
                        0x59, 0x92, 0xca, 0x29, 0xcc, 0x2c, 0xb1, 0x9b, 0xc6, 0x8e, 0xff, 0xf1,
                        0x29, 0xae, 0x21, 0xe9, 0x17,
                    ],
                },
                Signature {
                    index: 0xb2,
                    signature: [
                        0xa1, 0x45, 0x54, 0x14, 0xd5, 0x3a, 0x4f, 0xb0, 0xf1, 0xf4, 0xf6, 0xf5,
                        0x6b, 0x17, 0xc2, 0x52, 0x19, 0xe8, 0x68, 0x54, 0x73, 0x39, 0xde, 0xd2,
                        0xef, 0x5c, 0xca, 0xca, 0x0f, 0x42, 0x0d, 0x3c, 0x71, 0x64, 0x50, 0xc0,
                        0x2f, 0xf3, 0xf8, 0x70, 0xee, 0x52, 0xa8, 0x4a, 0xfb, 0x2a, 0x62, 0x4d,
                        0xeb, 0xc8, 0x1e, 0xa3, 0x38, 0x07, 0x78, 0x67, 0x7f, 0x4b, 0x96, 0xa0,
                        0x54, 0xc0, 0x66, 0x7d, 0xe7,
                    ],
                },
            ]),
            timestamp: 0x2db5_98b3,
            nonce: 0x0861_20c4,
            emitter_chain: 0x247b,
            emitter_address: [
                0x8b, 0xc0, 0x03, 0x0d, 0xe2, 0x50, 0x96, 0xcc, 0x48, 0xa8, 0xe7, 0xd7, 0x17, 0x05,
                0x6f, 0x9c, 0xe8, 0xe8, 0x0c, 0x12, 0x0d, 0x05, 0x02, 0xed, 0x4c, 0xc9, 0x51, 0xb4,
                0x9c, 0xe3, 0xc7, 0x94,
            ],
            sequence: 0xcc2b_6c34_eda9_89c1,
            consistency_level: 0x0d,
            map: BTreeMap::from([(0x35845d1a, 0x25ff53af), (0x543596f3, 0x58373435)]),
            payload: GovernancePacket {
                module: [
                    0x50, 0x06, 0x58, 0xff, 0xff, 0xae, 0x1a, 0xdd, 0x07, 0xbc, 0xcf, 0x34, 0x10,
                    0x6c, 0xa3, 0xbb, 0x14, 0x40, 0x25, 0xe1, 0x8f, 0x1a, 0xa0, 0x39, 0x7b, 0x12,
                    0x5a, 0x03, 0x58, 0x6f, 0xe1, 0x88,
                ],
                action: Action::ContractUpgrade,
                chain: Chain::Solana,
            },
        };
        let payload = &[0x3d, 0xab, 0x45, 0xaf, 0x7a, 0x6e, 0x9f, 0x7b];

        let mut buf = to_vec(&vaa).unwrap();
        buf.extend_from_slice(payload);

        let (actual, governance_payload) = from_slice_with_payload(&buf).unwrap();

        assert_eq!(vaa, actual);

        match actual.payload.action {
            Action::ContractUpgrade => {
                let expected = 0x3dab_45af_7a6e_9f7b;
                let msg: ContractUpgrade = from_slice(governance_payload).unwrap();
                assert_eq!(expected, msg.new_contract);
            }
            _ => panic!("Unexpected action: {:?}", actual.payload.action),
        }
    }
}
