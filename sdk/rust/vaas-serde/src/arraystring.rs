//! A module for serializing/deserializing a `BString` as a fixed-width 32 byte array.

use std::{convert::identity, fmt, iter::repeat_n};

use bstr::BString;
use serde::{
    de::{Error as DeError, SeqAccess, Visitor},
    ser::{Error as SerError, SerializeTuple},
    Deserializer, Serializer,
};

pub fn serialize<T, S>(value: T, serializer: S) -> Result<S::Ok, S::Error>
where
    T: AsRef<[u8]>,
    S: Serializer,
{
    let v = value.as_ref();
    let l = v.len();
    if l > 32 {
        return Err(S::Error::custom(format_args!(
            "value is too large ({l} bytes); max 32",
        )));
    }

    let mut tup = serializer.serialize_tuple(32)?;
    for e in repeat_n(&0u8, 32 - l).chain(v) {
        tup.serialize_element(e)?;
    }

    tup.end()
}

struct ArrayStringVisitor;
impl<'de> Visitor<'de> for ArrayStringVisitor {
    type Value = BString;

    fn expecting(&self, f: &mut fmt::Formatter) -> fmt::Result {
        f.write_str("an array of 32 bytes")
    }

    fn visit_seq<A>(self, mut seq: A) -> Result<Self::Value, A::Error>
    where
        A: SeqAccess<'de>,
    {
        let mut buf = Vec::with_capacity(32);

        for i in 0..32 {
            let e = seq
                .next_element()
                .map(|e| e.ok_or_else(|| A::Error::invalid_length(i, &self)))
                .and_then(identity)?; // TODO: Replace with Result::flatten once stabilized.

            if e == 0 && buf.is_empty() {
                // Skip all leading zeroes.
                continue;
            }

            buf.push(e);
        }

        Ok(BString::from(buf))
    }
}

pub fn deserialize<'de, D>(deserializer: D) -> Result<BString, D::Error>
where
    D: Deserializer<'de>,
{
    deserializer.deserialize_tuple(32, ArrayStringVisitor)
}

#[cfg(test)]
mod test {
    use bstr::BString;
    use serde::{Deserialize, Serialize};

    #[derive(Serialize, Deserialize, PartialEq, Eq, Debug, Clone)]
    #[repr(transparent)]
    struct MyString(#[serde(with = "super")] BString);

    #[test]
    fn end_to_end() {
        let v = "45825ca36ef7628727b83f8a409a08ad";
        let zeroes = [0u8; 32];

        for i in 0..=32 {
            let expected = MyString(v[i..].into());
            let buf = serde_wormhole::to_vec(&expected).unwrap();

            assert_eq!(&zeroes[..i], &buf[..i]);
            assert_eq!(v.as_bytes()[i..], buf[i..]);

            let actual = serde_wormhole::from_slice(&buf).unwrap();
            assert_eq!(expected, actual);
        }
    }

    #[test]
    fn value_too_large() {
        let v = MyString("61ddb8ede2a3f0cec9b550ef150c08096280d0480493365f".into());
        let _ = serde_wormhole::to_vec(&v)
            .expect_err("successfully serialized string longer than 32 bytes");
    }

    #[test]
    fn buffer_too_small() {
        let b = [0u8; 16];

        let _ = serde_wormhole::from_slice::<MyString>(&b)
            .expect_err("successfully deserialized string from a buffer that's too small");
    }
}
