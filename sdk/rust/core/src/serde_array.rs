use std::{fmt, mem::MaybeUninit};

use serde::{
    de::{Error, SeqAccess, Visitor},
    ser::SerializeTuple,
    Deserializer, Serializer,
};

pub fn serialize<const N: usize, S>(value: &[u8; N], serializer: S) -> Result<S::Ok, S::Error>
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
        write!(formatter, "an array of length {N}")
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
