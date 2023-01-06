//! This crate provides a mechanism to convert arbitrary rust structs into `cosmwasm_std::Event`s
//! via that struct's `Serialize` impl.

use cosmwasm_std::Event;
use serde::Serialize;

mod error;
mod ser;

pub use error::Error;
use ser::Serializer;

/// Convert `value` into a `cosmwasm_std::Event` via its `Serialize` impl.
///
/// `value` must serialize as exactly one regular struct, a unit struct, a struct variant of an
/// enum, or an `Option` that contains a value that serializes as one of of the previous 3 types.
/// Attempting to convert any other type (including `Option::None`) to an `Event` will return
/// `Error::NotAStruct`.
///
/// The name of the struct will become the type of the returned event while the fields of the struct
/// will become the event attributes.  In the case of a struct variant of an enum, the type of the
/// event will be "{name}::{variant}".  Field values are encoded using the `serde-json-wasm` crate
/// and field types may be any type supported by that crate.
///
/// # Examples
///
/// ```
/// # fn example() -> Result<(), cw_transcode::Error> {
/// #    use cosmwasm_std::{Binary, Event};
/// #    use serde::Serialize;
/// #
/// #    #[derive(Serialize)]
/// #    struct Nested {
/// #        f1: u32,
/// #        f2: String,
/// #    }
/// #
/// #    #[derive(Serialize)]
/// #    struct Example {
/// #        primitive: i64,
/// #        nested: Nested,
/// #        binary: Binary,
/// #    }
/// #
///     use cw_transcode::to_event;
///
///     let e = Example {
///         // Primitive values are supported.
///         primitive: 0x9a82f2b2865c2fdau64 as i64,
///
///         // Nested struct fields are encoded as json objects.
///         nested: Nested {
///             f1: 0xdbc7b8a1u32,
///             f2: "TEST".to_string(),
///         },
///
///         // Other types supported by `serde-json-wasm` are also fine.
///         binary: Binary(vec![
///             0xdb, 0x78, 0x97, 0x0e, 0xf2, 0x55, 0x97, 0xf4, 0x66, 0x11, 0x91, 0x0f, 0x50, 0x57,
///             0xb8, 0xe7,
///         ]),
///     };
///
///     let evt = to_event(&e)?;
///
///     let expected = Event::new("Example")
///         .add_attribute("primitive", "-7313015996323975206")
///         .add_attribute("nested", r#"{"f1":3687299233,"f2":"TEST"}"#)
///         .add_attribute("binary", "\"23iXDvJVl/RmEZEPUFe45w==\"");
///
///     assert_eq!(expected, evt);
/// #
/// #    Ok(())
/// # }
/// #
/// # example().unwrap();
/// ```
pub fn to_event<T: Serialize + ?Sized>(value: &T) -> Result<Event, Error> {
    let mut s = Serializer::new();
    value.serialize(&mut s)?;
    s.finish().ok_or(Error::NoEvent)
}

#[cfg(test)]
mod test {
    use super::*;

    use std::{any::type_name, collections::BTreeMap};

    use cosmwasm_std::Binary;
    use serde_bytes::ByteBuf;

    #[test]
    fn unsupported() {
        macro_rules! test_unsupported {
            ($($ty:ty),* $(,)?) => {
                $(
                    let x: $ty = Default::default();
                    let name = type_name::<$ty>();
                    let err = to_event(&x).expect_err(name);
                    assert!(matches!(err, Error::NotAStruct), "{name}");
                )*
            }
        }

        test_unsupported!(
            i8, u8, i16, u16, i32, u32, i64, u64, i128, u128, f32, f64, char,
            String, Vec<u8>, ByteBuf, BTreeMap<u32, u32>, Option<u64>, (), (u32, f32, i64)
        );
    }

    #[test]
    fn basic() {
        #[derive(Serialize)]
        struct Transfer {
            tx_hash: Binary,
            timestamp: u32,
            nonce: u32,
            emitter_chain: u16,
            #[serde(with = "hex")]
            emitter_address: [u8; 32],
            sequence: u64,
            consistency_level: u8,
            payload: Binary,
        }

        let tx = Transfer {
            tx_hash: Binary(vec![
                0x82, 0xea, 0x25, 0x36, 0xc5, 0xd1, 0x67, 0x18, 0x30, 0xcb, 0x49, 0x12, 0x0f, 0x94,
                0x47, 0x9e, 0x34, 0xb5, 0x45, 0x96, 0xa8, 0xdd, 0x36, 0x9f, 0xbc, 0x26, 0x66, 0x66,
                0x7a, 0x76, 0x5f, 0x4b,
            ]),
            timestamp: 1672860466,
            nonce: 0,
            emitter_chain: 2,
            emitter_address: [
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x90,
                0xfb, 0x16, 0x72, 0x08, 0xaf, 0x45, 0x5b, 0xb1, 0x37, 0x78, 0x01, 0x63, 0xb7, 0xb7,
                0xa9, 0xa1, 0x0c, 0x16,
            ],
            sequence: 1672860466,
            consistency_level: 15,
            payload: Binary(vec![
                0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0xe0, 0xb6,
                0xb3, 0xa7, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x2d, 0x8b, 0xe6, 0xbf, 0x0b, 0xaa, 0x74, 0xe0, 0xa9, 0x07, 0x01,
                0x66, 0x79, 0xca, 0xe9, 0x19, 0x0e, 0x80, 0xdd, 0x0a, 0x00, 0x02, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc1, 0x08, 0x20, 0x98, 0x3f,
                0x33, 0x45, 0x6c, 0xe7, 0xbe, 0xb3, 0xa0, 0x46, 0xf5, 0xa8, 0x3f, 0xa3, 0x4f, 0x02,
                0x7d, 0x0c, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            ]),
        };

        let expected = Event::new("Transfer")
            .add_attribute(
                "tx_hash",
                "\"guolNsXRZxgwy0kSD5RHnjS1RZao3TafvCZmZnp2X0s=\"",
            )
            .add_attribute("timestamp", "1672860466")
            .add_attribute("nonce", "0")
            .add_attribute("emitter_chain", "2")
            .add_attribute(
                "emitter_address",
                "\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\"",
            )
            .add_attribute("sequence", "1672860466")
            .add_attribute("consistency_level", "15")
            .add_attribute("payload", "\"AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\"");

        let actual = to_event(&tx).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn option() {
        #[derive(Serialize)]
        struct Data {
            a: u32,
            b: String,
        }

        let d = Some(Data {
            a: 17,
            b: "BEEF".into(),
        });

        let expected = Event::new("Data")
            .add_attribute("a", "17")
            .add_attribute("b", "\"BEEF\"");

        let actual = to_event(&d).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn unit_struct() {
        #[derive(Serialize)]
        struct MyEvent;

        let expected = Event::new("MyEvent");
        let actual = to_event(&MyEvent).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn enum_variants() {
        #[derive(Serialize)]
        enum MyEvent {
            A,
            B(u32),
            C { f1: u64, f2: String },
            D(u16, u16, u32),
        }

        // Unit variants.
        {
            let expected = Event::new("MyEvent::A");
            let actual = to_event(&MyEvent::A).unwrap();
            assert_eq!(expected, actual);
        }

        // Newtype variants.
        {
            let err = to_event(&MyEvent::B(19)).unwrap_err();
            assert!(matches!(err, Error::NotAStruct));
        }

        // Struct variants.
        {
            let c = MyEvent::C {
                f1: 500,
                f2: "test struct variant".into(),
            };
            let expected = Event::new("MyEvent::C")
                .add_attribute("f1", "500")
                .add_attribute("f2", "\"test struct variant\"");
            let actual = to_event(&c).unwrap();
            assert_eq!(expected, actual);
        }

        // Tuple variants.
        {
            let err = to_event(&MyEvent::D(37, 1926, 189174)).unwrap_err();
            assert!(matches!(err, Error::NotAStruct));
        }
    }

    #[test]
    fn nested() {
        #[derive(Serialize)]
        struct Nested {
            a: u32,
            b: Binary,
        }

        #[derive(Serialize)]
        struct Outer {
            nested: Nested,
            c: String,
        }

        let o = Outer {
            nested: Nested {
                a: 0xfeb42045,
                b: Binary(vec![
                    0x11, 0x7d, 0x83, 0xa4, 0x06, 0xbf, 0x3e, 0x50, 0xe1, 0xaa, 0x19, 0x89, 0xc3,
                    0x00, 0xea, 0xaf,
                ]),
            },
            c: "TEST".into(),
        };

        let expected = Event::new("Outer")
            .add_attribute(
                "nested",
                r#"{"a":4273217605,"b":"EX2DpAa/PlDhqhmJwwDqrw=="}"#,
            )
            .add_attribute("c", "\"TEST\"");
        let actual = to_event(&o).unwrap();
        assert_eq!(expected, actual);
    }
}
