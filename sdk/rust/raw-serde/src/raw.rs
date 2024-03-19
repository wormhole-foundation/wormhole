use std::{fmt, ops::Deref};

use base64::display::Base64Display;
use serde::{
    de::{Error as DeError, SeqAccess, Visitor},
    Deserialize, Deserializer, Serialize, Serializer,
};
use serde_bytes::Bytes;

use crate::Error;

pub(crate) const TOKEN: &str = "$serde_wormhole::private::RawMessage";

/// Reference to a range of bytes in the input data.
///
/// A `RawMessage` can be used to defer parsing parts of the input data until later, or to avoid
/// parsing it at all if it needs to be passed on verbatim to a different output object.
///
/// When used to deserialize data in the wormhole data format, `RawMessage` will consume all the
/// remaining data in the input since the wormhole wire format is not self-describing.  However when
/// used with self-describing formats like JSON, `RawMessage` will expect either a sequence of bytes
/// or a base64-encoded string.
///
/// When serializing, a `RawMessage` will either serialize to a base64-encoded string if the data
/// format is human readable (like JSON) or will forward the raw bytes to the serializer if not.
///
/// # Examples
///
/// Defer parsing the payload of a VAA body:
///
/// ```
/// # fn example() -> Result<(), serde_wormhole::Error> {
/// #     let data = [
/// #         0x62, 0xb9, 0xf7, 0x91, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
/// #         0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf1, 0x9a, 0x2a, 0x01, 0xb7, 0x05, 0x19, 0xf6,
/// #         0x7a, 0xdb, 0x30, 0x9a, 0x99, 0x4e, 0xc8, 0xc6, 0x9a, 0x96, 0x7e, 0x8b, 0x00, 0x00, 0x00,
/// #         0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x46, 0x72, 0x6f, 0x6d, 0x3a, 0x20, 0x65, 0x76, 0x6d,
/// #         0x30, 0x5c, 0x6e, 0x4d, 0x73, 0x67, 0x3a, 0x20, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x57,
/// #         0x6f, 0x72, 0x6c, 0x64, 0x21,
/// #     ];
/// #
///       use serde::{Serialize, Deserialize};
///       use serde_wormhole::{from_slice, RawMessage};
///
///       #[derive(Serialize, Deserialize, Debug)]
///       struct Body<'a> {
///           timestamp: u32,
///           nonce: u32,
///           emitter_chain: u16,
///           emitter_address: [u8; 32],
///           sequence: u64,
///           consistency_level: u8,
///           #[serde(borrow)]
///           payload: &'a RawMessage,
///       }
///  
///       let body = from_slice::<Body>(&data)?;
///       assert_eq!(b"From: evm0\\nMsg: Hello World!", body.payload.get());
/// #
/// #     Ok(())
/// # }
/// #
/// # example().unwrap();
/// ```
///
/// # Ownership
///
/// The typical usage of `RawMessage` will be in its borrowed form:
///
/// ```
/// # use serde::Deserialize;
/// # use serde_wormhole::RawMessage;
/// #
/// #[derive(Deserialize)]
/// struct MyStruct<'a> {
///     #[serde(borrow)]
///     raw_message: &'a RawMessage,
/// }
/// ```
///
/// The borrowed form is suitable for use with `serde_wormhole::from_slice` because it supports
/// borrowing from the input data without memory allocation.  If the value is encoded as a string,
/// deserializing to the borrowed form may or may not succeed depending on the deserializer
/// implementation.  In the case where the deserialization is successful, the contents of the string
/// will not be interpreted in any way and the `RawMessage` will simply contain the raw bytes of
/// the input string.  This may have unexpected consequences (such as the bytes being base64-encoded
/// if the `RawMessage` is re-serialized, potentially leading to double-encoding).  In general, you
/// should only use the borrowed form if you know the input data contains raw bytes.  Otherwise, the
/// boxed form is a safer choice.
///
/// When deserializing through `serde_wormhole::from_reader` or when the value is encoded as a
/// base64 string, it is necessary to use the boxed form.  This involves either copying the data
/// from the IO stream or decoding the base64 string and then storing it in memory.
///
/// ```
/// # use serde::Deserialize;
/// # use serde_wormhole::RawMessage;
/// #
/// #[derive(Deserialize)]
/// struct MyStruct {
///     raw_message: Box<RawMessage>,
/// }
/// ```
#[repr(transparent)]
#[derive(PartialEq, Eq, PartialOrd, Ord)]
pub struct RawMessage {
    bytes: [u8],
}

impl RawMessage {
    const fn from_borrowed(b: &[u8]) -> &Self {
        // Safety: repr(transparent) guarantees that `RawMessage` and `[u8]` have the same layout
        // and ABI.
        unsafe { &*(b as *const [u8] as *const RawMessage) }
    }

    fn from_owned(b: Box<[u8]>) -> Box<Self> {
        #[cfg(debug_assertions)]
        {
            use std::alloc::Layout;

            let a = Layout::for_value::<[u8]>(&b);
            let b = Layout::for_value::<RawMessage>(Self::from_borrowed(&b));
            debug_assert_eq!(a, b);
        }

        // Safety: repr(transparent) guarantees that `RawMessage` and `[u8]` have the same layout
        // and ABI.
        unsafe { Box::from_raw(Box::into_raw(b) as *mut Self) }
    }

    fn into_owned(self: Box<Self>) -> Box<[u8]> {
        #[cfg(debug_assertions)]
        {
            use std::alloc::Layout;

            let a = Layout::for_value::<RawMessage>(&self);
            let b = Layout::for_value::<[u8]>(&self.bytes);
            debug_assert_eq!(a, b);
        }

        // Safety: repr(transparent) guarantees that `RawMessage` and `[u8]` have the same layout
        // and ABI.
        unsafe { Box::from_raw(Box::into_raw(self) as *mut [u8]) }
    }

    /// Create a new borrowed `RawMessage` from an existing `&[u8]`.
    pub const fn new(b: &[u8]) -> &Self {
        Self::from_borrowed(b)
    }

    /// Access the raw bytes underlying a `RawMessage`.
    pub const fn get(&self) -> &[u8] {
        &self.bytes
    }
}

impl<'a> From<&'a [u8]> for &'a RawMessage {
    fn from(value: &'a [u8]) -> Self {
        RawMessage::new(value)
    }
}

impl<'a> From<&'a RawMessage> for &'a [u8] {
    fn from(value: &'a RawMessage) -> Self {
        &value.bytes
    }
}

impl From<Vec<u8>> for Box<RawMessage> {
    fn from(value: Vec<u8>) -> Self {
        RawMessage::from_owned(value.into_boxed_slice())
    }
}

impl From<Box<[u8]>> for Box<RawMessage> {
    fn from(value: Box<[u8]>) -> Self {
        RawMessage::from_owned(value)
    }
}

impl From<Box<RawMessage>> for Box<[u8]> {
    fn from(value: Box<RawMessage>) -> Self {
        value.into_owned()
    }
}

impl AsRef<[u8]> for RawMessage {
    fn as_ref(&self) -> &[u8] {
        &self.bytes
    }
}

impl Deref for RawMessage {
    type Target = [u8];

    fn deref(&self) -> &Self::Target {
        &self.bytes
    }
}

impl fmt::Debug for RawMessage {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_tuple("RawMessage")
            .field(&format_args!("{:?}", &self.bytes))
            .finish()
    }
}

impl fmt::Display for RawMessage {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{}",
            Base64Display::with_config(&self.bytes, base64::STANDARD)
        )
    }
}

impl ToOwned for RawMessage {
    type Owned = Box<RawMessage>;

    fn to_owned(&self) -> Self::Owned {
        RawMessage::from_owned(self.bytes.to_owned().into_boxed_slice())
    }
}

impl Clone for Box<RawMessage> {
    fn clone(&self) -> Self {
        (**self).to_owned()
    }
}

impl Default for Box<RawMessage> {
    fn default() -> Self {
        RawMessage::from_owned(Default::default())
    }
}

impl Serialize for RawMessage {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        if serializer.is_human_readable() {
            let bytes = base64::encode(&self.bytes);
            serializer.serialize_newtype_struct(TOKEN, &bytes)
        } else {
            serializer.serialize_newtype_struct(TOKEN, Bytes::new(&self.bytes))
        }
    }
}

impl<'a, 'de: 'a> Deserialize<'de> for &'a RawMessage {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct ReferenceVisitor;

        impl<'de> Visitor<'de> for ReferenceVisitor {
            type Value = &'de RawMessage;

            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                formatter.write_str("a borrowed byte slice")
            }

            fn visit_newtype_struct<D>(self, deserializer: D) -> Result<Self::Value, D::Error>
            where
                D: Deserializer<'de>,
            {
                deserializer.deserialize_bytes(self)
            }

            fn visit_borrowed_bytes<E>(self, v: &'de [u8]) -> Result<Self::Value, E>
            where
                E: DeError,
            {
                Ok(RawMessage::from_borrowed(v))
            }
        }

        deserializer.deserialize_newtype_struct(TOKEN, ReferenceVisitor)
    }
}

impl<'de> Deserialize<'de> for Box<RawMessage> {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        struct BoxedVisitor;

        impl<'de> Visitor<'de> for BoxedVisitor {
            type Value = Box<RawMessage>;

            fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
                formatter.write_str("a byte slice or a base64-encoded string")
            }

            fn visit_str<E>(self, v: &str) -> Result<Self::Value, E>
            where
                E: DeError,
            {
                let v = base64::decode(v)
                    .map_err(|e| E::custom(format_args!("failed to decode base64: {e}")))?;
                Ok(RawMessage::from_owned(v.into_boxed_slice()))
            }

            fn visit_newtype_struct<D>(self, deserializer: D) -> Result<Self::Value, D::Error>
            where
                D: Deserializer<'de>,
            {
                deserializer.deserialize_any(self)
            }

            fn visit_bytes<E>(self, v: &[u8]) -> Result<Self::Value, E>
            where
                E: DeError,
            {
                Ok(RawMessage::from_owned(v.to_owned().into_boxed_slice()))
            }

            fn visit_byte_buf<E>(self, v: Vec<u8>) -> Result<Self::Value, E>
            where
                E: DeError,
            {
                Ok(RawMessage::from_owned(v.into_boxed_slice()))
            }

            fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
            where
                A: SeqAccess<'de>,
            {
                let mut buf = Vec::with_capacity(seq.size_hint().unwrap_or(0));

                while let Some(b) = seq.next_element()? {
                    buf.push(b);
                }

                Ok(RawMessage::from_owned(buf.into_boxed_slice()))
            }
        }

        deserializer.deserialize_newtype_struct(TOKEN, BoxedVisitor)
    }
}

/// Convert a `T` into a boxed `RawMessage`.
pub fn to_raw_message<T: Serialize + ?Sized>(value: &T) -> Result<Box<RawMessage>, Error> {
    let bytes = crate::to_vec(value)?;
    Ok(RawMessage::from_owned(bytes.into_boxed_slice()))
}

#[cfg(test)]
mod test {
    use super::*;

    #[derive(Serialize, Deserialize, Debug, PartialEq)]
    struct MyStruct<P> {
        f1: u32,
        f2: u16,
        payload: P,
    }

    #[test]
    fn borrowed() {
        let data = [
            0x5b, 0x4a, 0x55, 0xca, 0x80, 0x53, 0xfe, 0x25, 0x6d, 0xdc, 0xb3, 0x3b, 0x8d, 0x38,
            0xf7, 0x1b,
        ];

        let expected = MyStruct {
            f1: 0x5b4a55ca,
            f2: 0x8053,
            payload: RawMessage::from_borrowed(&data[6..]),
        };

        let actual = crate::from_slice::<MyStruct<&RawMessage>>(&data).unwrap();
        assert_eq!(expected, actual);
        assert_eq!(&data[..], crate::to_vec(&expected).unwrap());
    }

    #[test]
    fn owned() {
        let data = [
            0x5b, 0x4a, 0x55, 0xca, 0x80, 0x53, 0xfe, 0x25, 0x6d, 0xdc, 0xb3, 0x3b, 0x8d, 0x38,
            0xf7, 0x1b,
        ];

        let expected = MyStruct {
            f1: 0x5b4a55ca,
            f2: 0x8053,
            payload: Box::<RawMessage>::from(data[6..].to_vec()),
        };

        let actual = crate::from_slice::<MyStruct<Box<RawMessage>>>(&data).unwrap();
        assert_eq!(expected, actual);
        assert_eq!(&data[..], crate::to_vec(&expected).unwrap());
    }

    #[test]
    fn json_string() {
        #[derive(Serialize, Deserialize, Debug, PartialEq)]
        struct MyStruct {
            f1: u32,
            payload: Box<RawMessage>,
            f2: u16,
        }
        let data = r#"{"f1":1531598282,"payload":"/iVt3LM7jTj3Gw==","f2":32851}"#;

        let expected = MyStruct {
            f1: 0x5b4a55ca,
            payload: Box::<RawMessage>::from(vec![
                0xfe, 0x25, 0x6d, 0xdc, 0xb3, 0x3b, 0x8d, 0x38, 0xf7, 0x1b,
            ]),
            f2: 0x8053,
        };

        let actual = serde_json::from_str(data).unwrap();
        assert_eq!(expected, actual);
        assert_eq!(data, serde_json::to_string(&expected).unwrap());
    }

    #[test]
    fn json_sequence() {
        let data = r#"{"f1":1531598282,"f2":32851,"payload":[223,35,191,255,175]}"#;
        let expected = MyStruct {
            f1: 0x5b4a55ca,
            f2: 0x8053,
            payload: Box::<RawMessage>::from(vec![223, 35, 191, 255, 175]),
        };

        let actual = serde_json::from_str(data).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn json_reference() {
        #[derive(Serialize, Deserialize, Debug, PartialEq)]
        struct Referenced<'a> {
            #[serde(borrow)]
            raw_message: &'a RawMessage,
        }

        let data = r#"{"raw_message":"/iVt3LM7jTj3Gw=="}"#;
        let expected = Referenced {
            raw_message: RawMessage::from_borrowed(&[
                0x2f, 0x69, 0x56, 0x74, 0x33, 0x4c, 0x4d, 0x37, 0x6a, 0x54, 0x6a, 0x33, 0x47, 0x77,
                0x3d, 0x3d,
            ]),
        };

        // This works because serde_json will forward the raw bytes of the string to the visitor
        // but the content won't be decoded.  This *would not* work if the value contained invalid
        // UTF-8.
        let actual = serde_json::from_str::<Referenced>(data).unwrap();
        assert_eq!(expected, actual);

        // Serializing will re-encode the bytes so we'll get a different result.
        assert_eq!(
            r#"{"raw_message":"L2lWdDNMTTdqVGozR3c9PQ=="}"#.as_bytes(),
            serde_json::to_vec(&expected).unwrap()
        );
    }
}
