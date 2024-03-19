use std::{
    convert::TryFrom,
    mem::{self, size_of},
};

use serde::de::{
    self, value::BorrowedBytesDeserializer, DeserializeSeed, EnumAccess, Error as DeError,
    IntoDeserializer, MapAccess, SeqAccess, VariantAccess, Visitor,
};

use crate::error::Error;

/// A struct that deserializes the VAA payload wire format into rust values.
pub struct Deserializer<'de> {
    input: &'de [u8],
}

impl<'de> Deserializer<'de> {
    /// Construct a new instance of `Deserializer` from `input`.
    pub fn new(input: &'de [u8]) -> Self {
        Self { input }
    }

    /// Should be called once the value has been fully deserialized.  Returns any data left in the
    /// input buffer after deserialization.
    pub fn end(self) -> &'de [u8] {
        self.input
    }
}

// This has to be a macro because `<type>::from_be_bytes` is not a trait function so there is no
// trait bound that we can use in a generic function.
macro_rules! deserialize_be_number {
    ($self:ident, $ty:ty) => {{
        const LEN: usize = size_of::<$ty>();
        if $self.input.len() < LEN {
            return Err(Error::Eof);
        }

        let (data, rem) = $self.input.split_at(LEN);
        let mut buf = [0u8; LEN];
        buf.copy_from_slice(data);
        $self.input = rem;

        <$ty>::from_be_bytes(buf)
    }};
}

impl<'de, 'a> de::Deserializer<'de> for &'a mut Deserializer<'de> {
    type Error = Error;

    fn deserialize_any<V>(self, _: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        Err(Error::DeserializeAnyNotSupported)
    }

    fn deserialize_ignored_any<V>(self, _: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        Err(Error::DeserializeAnyNotSupported)
    }

    fn deserialize_bool<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let v = deserialize_be_number!(self, u8);
        match v {
            0 => visitor.visit_bool(false),
            1 => visitor.visit_bool(true),
            v => Err(Error::custom(format_args!(
                "invalid value: {v}, expected a 0 or 1"
            ))),
        }
    }

    fn deserialize_i8<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_i8(deserialize_be_number!(self, i8))
    }

    fn deserialize_i16<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_i16(deserialize_be_number!(self, i16))
    }

    fn deserialize_i32<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_i32(deserialize_be_number!(self, i32))
    }

    fn deserialize_i64<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_i64(deserialize_be_number!(self, i64))
    }

    fn deserialize_i128<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_i128(deserialize_be_number!(self, i128))
    }

    fn deserialize_u8<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_u8(deserialize_be_number!(self, u8))
    }

    fn deserialize_u16<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_u16(deserialize_be_number!(self, u16))
    }

    fn deserialize_u32<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_u32(deserialize_be_number!(self, u32))
    }

    fn deserialize_u64<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_u64(deserialize_be_number!(self, u64))
    }

    fn deserialize_u128<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_u128(deserialize_be_number!(self, u128))
    }

    fn deserialize_f32<V>(self, _: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        Err(Error::Unsupported)
    }

    fn deserialize_f64<V>(self, _: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        Err(Error::Unsupported)
    }

    fn deserialize_char<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let v = deserialize_be_number!(self, u32);
        char::try_from(v)
            .map_err(|e| Error::custom(format_args!("invalid value {v}: {e}")))
            .and_then(|v| visitor.visit_char(v))
    }

    fn deserialize_str<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let len = usize::from(deserialize_be_number!(self, u8));

        if self.input.len() < len {
            return Err(Error::Eof);
        }

        let (data, rem) = self.input.split_at(len);
        self.input = rem;

        std::str::from_utf8(data)
            .map_err(Error::custom)
            .and_then(|s| visitor.visit_borrowed_str(s))
    }

    #[inline]
    fn deserialize_string<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        self.deserialize_str(visitor)
    }

    fn deserialize_bytes<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let len = usize::from(deserialize_be_number!(self, u8));

        if self.input.len() < len {
            return Err(Error::Eof);
        }

        let (data, rem) = self.input.split_at(len);
        self.input = rem;

        visitor.visit_borrowed_bytes(data)
    }

    #[inline]
    fn deserialize_byte_buf<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        self.deserialize_bytes(visitor)
    }

    #[inline]
    fn deserialize_option<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        // There are no optional values in this data format.
        visitor.visit_some(self)
    }

    #[inline]
    fn deserialize_unit<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_unit()
    }

    #[inline]
    fn deserialize_unit_struct<V>(
        self,
        _name: &'static str,
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_unit()
    }

    #[inline]
    fn deserialize_newtype_struct<V>(
        self,
        name: &'static str,
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        if name == crate::raw::TOKEN {
            let rem = mem::take(&mut self.input);
            visitor.visit_newtype_struct(BorrowedBytesDeserializer::new(rem))
        } else {
            visitor.visit_newtype_struct(self)
        }
    }

    fn deserialize_seq<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let len = usize::from(deserialize_be_number!(self, u8));
        visitor.visit_seq(BoundedSequence::new(self, len))
    }

    #[inline]
    fn deserialize_tuple<V>(self, len: usize, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_seq(BoundedSequence::new(self, len))
    }

    #[inline]
    fn deserialize_tuple_struct<V>(
        self,
        _name: &'static str,
        len: usize,
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_seq(BoundedSequence::new(self, len))
    }

    fn deserialize_map<V>(self, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let len = usize::from(deserialize_be_number!(self, u8));
        visitor.visit_map(BoundedSequence::new(self, len))
    }

    #[inline]
    fn deserialize_struct<V>(
        self,
        _name: &'static str,
        fields: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_seq(BoundedSequence::new(self, fields.len()))
    }

    fn deserialize_enum<V>(
        self,
        _name: &'static str,
        _variants: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        let variant = deserialize_be_number!(self, u8);
        visitor.visit_enum(Enum { de: self, variant })
    }

    fn deserialize_identifier<V>(self, _: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        Err(Error::Unsupported)
    }
}

impl<'de, 'a> VariantAccess<'de> for &'a mut Deserializer<'de> {
    type Error = Error;

    #[inline]
    fn unit_variant(self) -> Result<(), Self::Error> {
        Ok(())
    }

    #[inline]
    fn newtype_variant_seed<T>(self, seed: T) -> Result<T::Value, Self::Error>
    where
        T: DeserializeSeed<'de>,
    {
        seed.deserialize(self)
    }

    #[inline]
    fn tuple_variant<V>(self, len: usize, visitor: V) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_seq(BoundedSequence::new(self, len))
    }

    #[inline]
    fn struct_variant<V>(
        self,
        fields: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value, Self::Error>
    where
        V: Visitor<'de>,
    {
        visitor.visit_seq(BoundedSequence::new(self, fields.len()))
    }
}

struct BoundedSequence<'de, 'a> {
    de: &'a mut Deserializer<'de>,
    count: usize,
}

impl<'de, 'a> BoundedSequence<'de, 'a> {
    fn new(de: &'a mut Deserializer<'de>, count: usize) -> Self {
        Self { de, count }
    }
}

impl<'de, 'a> SeqAccess<'de> for BoundedSequence<'de, 'a> {
    type Error = Error;

    fn next_element_seed<T>(&mut self, seed: T) -> Result<Option<T::Value>, Self::Error>
    where
        T: DeserializeSeed<'de>,
    {
        if self.count == 0 {
            return Ok(None);
        }

        self.count -= 1;
        seed.deserialize(&mut *self.de).map(Some)
    }

    #[inline]
    fn size_hint(&self) -> Option<usize> {
        Some(self.count)
    }
}

impl<'de, 'a> MapAccess<'de> for BoundedSequence<'de, 'a> {
    type Error = Error;

    fn next_key_seed<K>(&mut self, seed: K) -> Result<Option<K::Value>, Self::Error>
    where
        K: DeserializeSeed<'de>,
    {
        if self.count == 0 {
            return Ok(None);
        }

        self.count -= 1;
        seed.deserialize(&mut *self.de).map(Some)
    }

    #[inline]
    fn next_value_seed<V>(&mut self, seed: V) -> Result<V::Value, Self::Error>
    where
        V: DeserializeSeed<'de>,
    {
        seed.deserialize(&mut *self.de)
    }

    #[inline]
    fn size_hint(&self) -> Option<usize> {
        Some(self.count)
    }
}

/// Tells serde which enum variant it should deserialize. Enums are encoded in the byte stream as a
/// `u8` followed by the data for the variant but unfortunately, serde doesn't currently support
/// integer tags (see https://github.com/serde-rs/serde/issues/745). Instead we format the integer
/// into its string representation and have serde use that to determine the enum variant. This
/// requires using `#[serde(rename = "<integer tag>")]` on all enum variants.
///
/// # Examples
///
/// ```
/// use std::borrow::Cow;
///
/// use serde::{Deserialize, Serialize};
///
/// #[derive(Serialize, Deserialize)]
/// enum TestEnum<'a> {
///     #[serde(rename = "19")]
///     Unit,
///     #[serde(rename = "235")]
///     NewType(u64),
///     #[serde(rename = "179")]
///     Tuple(u32, u64, Vec<u16>),
///     #[serde(rename = "97")]
///     Struct {
///         #[serde(borrow, with = "serde_bytes")]
///         data: Cow<'a, [u8]>,
///         footer: u32,
///     },
/// }
/// #
/// # assert!(matches!(serde_wormhole::from_slice(&[19]).unwrap(), TestEnum::Unit));
/// ```
struct Enum<'de, 'a> {
    de: &'a mut Deserializer<'de>,
    variant: u8,
}

impl<'de, 'a> EnumAccess<'de> for Enum<'de, 'a> {
    type Error = Error;
    type Variant = &'a mut Deserializer<'de>;

    fn variant_seed<V>(self, seed: V) -> Result<(V::Value, Self::Variant), Self::Error>
    where
        V: DeserializeSeed<'de>,
    {
        let mut buf = itoa::Buffer::new();
        seed.deserialize(buf.format(self.variant).into_deserializer())
            .map(|v| (v, self.de))
    }
}

#[cfg(test)]
mod tests {
    use std::{
        borrow::Cow,
        collections::BTreeMap,
        io::{Cursor, Write},
        iter::FromIterator,
        mem::size_of,
    };

    use serde::{Deserialize, Serialize};

    use crate::{from_slice, Error};

    #[test]
    fn empty_input() {
        let e = from_slice::<u8>(&[]).expect_err("empty buffer deserialized");
        assert!(matches!(e, Error::Eof))
    }

    #[test]
    fn trailing_data() {
        macro_rules! check {
                ($buf:ident, $ty:ty) => {
                    let e = from_slice::<$ty>(&$buf)
                        .expect_err("deserialized with trailing data");
                    assert!(matches!(e, Error::TrailingData));
                };
                ($buf:ident, $($ty:ty),*) => {
                    $(
                        check!($buf, $ty);
                    )*
                };
            }
        let buf = 0x9ab0_8c9f_8462_2f63u64.to_be_bytes();
        check!(buf, i8, i16, i32, u8, u16, u32);
    }

    #[test]
    fn bool() {
        let v: bool = from_slice(&[0u8]).unwrap();
        assert!(!v);

        let v: bool = from_slice(&[1u8]).unwrap();
        assert!(v);

        for v in 2..=u8::MAX {
            from_slice::<bool>(&[v]).unwrap_err();
        }
    }

    #[test]
    fn integers() {
        macro_rules! check {
    	          ($v:ident, $ty:ty) => {
                    // Casting an integer from a larger width to a smaller width will truncate the
                    // upper bits.
                    let expected = $v as $ty;
                    let buf = expected.to_be_bytes();

                    let actual: $ty = from_slice(&buf).expect("failed to deserialize integer");
                    assert_eq!(expected, actual);
    	          };
                ($v:ident, $($ty:ty),*) => {
                    $(
                        check!($v, $ty);
                    )*
                };
            }

        // Value randomly generated from `dd if=/dev/urandom | xxd -p -l 16`.
        let v = 0x84f2_e24f_2e8a_734e_5a5f_def6_c597_f232u128;
        check!(v, i8, i16, i32, i64, i128, u8, u16, u32, u64, u128);
    }

    #[test]
    fn char() {
        let chars = ['\u{0065}', '\u{0301}'];
        let mut buf = [0u8; size_of::<u32>() * 2];
        let mut cursor = Cursor::new(&mut buf[..]);
        for c in chars {
            cursor.write_all(&u32::from(c).to_be_bytes()).unwrap();
        }

        let (c1, c2): (char, char) = from_slice(&buf).unwrap();
        let mut actual = String::new();
        actual.push(c1);
        actual.push(c2);
        assert_eq!("é", actual);
    }

    #[test]
    fn invalid_chars() {
        for c in 0xd800u32..0xe000 {
            from_slice::<char>(&c.to_be_bytes()).expect_err("deserialized invalid char");
        }

        // Spot check a few values in the higher range.
        let invalid = [
            0xa17c509eu32,
            0xb4ee4bc8u32,
            0x46055273u32,
            0x3d3bb5fbu32,
            0xeb82fddcu32,
            0xe2777604u32,
            0xe597554fu32,
            0x12aa069fu32,
        ];
        for c in invalid {
            from_slice::<char>(&c.to_be_bytes()).expect_err("deserialized invalid char");
        }
    }

    #[test]
    fn char_array() {
        let expected = "Löwe 老虎 Léopard";
        let mut buf = [0u8; size_of::<u32>() * 15];
        let mut cursor = Cursor::new(&mut buf[..]);
        for c in expected.chars().map(u32::from) {
            cursor.write_all(&c.to_be_bytes()).unwrap();
        }

        let actual = from_slice::<[char; 15]>(&buf)
            .map(String::from_iter)
            .unwrap();

        assert_eq!(expected, actual);
    }

    #[test]
    fn strings() {
        let expected = "Löwe 老虎 Léopard";
        let buf = IntoIterator::into_iter([expected.len() as u8])
            .chain(expected.as_bytes().iter().copied())
            .collect::<Vec<u8>>();

        let actual = from_slice::<String>(&buf).unwrap();
        assert_eq!(expected, actual);

        let actual = from_slice::<&str>(&buf).unwrap();
        assert_eq!(expected, actual);

        let actual = from_slice::<&str>(&[0]).unwrap();
        assert!(actual.is_empty());
    }

    #[test]
    fn maps() {
        let buf = [
            0x04, 0x78, 0x26, 0x20, 0x90, 0x48, 0x96, 0xd4, 0x18, 0x8b, 0xce, 0x62, 0xcd, 0x87,
            0x7a, 0x36, 0x1a, 0x4c, 0x5e, 0x4f, 0x65, 0x84, 0x76, 0xb3, 0x9c, 0x7e, 0xb3, 0xfa,
            0x99, 0x29, 0xf2, 0x8b, 0x7f,
        ];

        let expected = BTreeMap::from([
            (0x78262090u32, 0x4896d418u32),
            (0x8bce62cd, 0x877a361a),
            (0x4c5e4f65, 0x8476b39c),
            (0x7eb3fa99, 0x29f28b7f),
        ]);

        let actual = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);

        let actual = from_slice::<BTreeMap<u32, u32>>(&[0]).unwrap();
        assert!(actual.is_empty());
    }

    #[test]
    fn empty_bytes() {
        let buf = [0x0];
        let actual: Vec<u8> = from_slice(&buf).unwrap();
        assert!(actual.is_empty());

        let actual: &[u8] = from_slice(&buf).unwrap();
        assert!(actual.is_empty());
    }

    #[test]
    fn bytes() {
        let buf = [
            0x40, 0xc7, 0xe7, 0x8f, 0x91, 0x47, 0x32, 0xe0, 0x54, 0x4e, 0xde, 0x94, 0x27, 0xf4,
            0xa9, 0x95, 0xd5, 0x96, 0xbe, 0x38, 0xd4, 0xa8, 0xca, 0xdd, 0x2e, 0xec, 0x95, 0x8d,
            0xb3, 0x1a, 0xa3, 0x8a, 0x3b, 0xc2, 0xdb, 0x54, 0xac, 0x23, 0x85, 0xa7, 0xe8, 0x88,
            0x39, 0xcb, 0xa4, 0x83, 0xde, 0xc4, 0x33, 0x83, 0x10, 0xba, 0x39, 0x55, 0x63, 0x67,
            0xd9, 0x08, 0x19, 0xe2, 0x42, 0xf6, 0xc9, 0x5c, 0xe2,
        ];
        let expected = &buf[1..];

        let actual: Vec<u8> = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);

        let actual: &[u8] = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn max_bytes() {
        let buf = [
            0xff, 0x8a, 0xa0, 0x62, 0x4b, 0xf8, 0x5c, 0x2f, 0x71, 0x8b, 0xa2, 0xe4, 0x80, 0xbf,
            0xb0, 0x15, 0xe0, 0xa3, 0x7c, 0xd3, 0x81, 0x56, 0x0d, 0x25, 0x13, 0x63, 0x23, 0xa1,
            0x0f, 0x84, 0x7f, 0x3e, 0xed, 0x3a, 0xe1, 0xe2, 0x8e, 0x20, 0x33, 0x42, 0x83, 0x89,
            0xa9, 0x0d, 0xe6, 0x58, 0xa5, 0xb4, 0x64, 0x60, 0x0f, 0x8f, 0xdf, 0x51, 0xd1, 0x00,
            0x9d, 0x4b, 0x6e, 0x42, 0x04, 0x8b, 0xa2, 0xc8, 0x14, 0xed, 0x4f, 0x46, 0x64, 0xf5,
            0xfd, 0xa6, 0xb2, 0x85, 0x63, 0x60, 0xa6, 0xb7, 0xd8, 0xed, 0x1a, 0xfd, 0x3f, 0x99,
            0x6b, 0x3c, 0x85, 0xfe, 0x09, 0x04, 0xab, 0x9f, 0x56, 0xfa, 0x9f, 0x80, 0xd5, 0x93,
            0x94, 0xa3, 0xc6, 0x62, 0xa8, 0x0e, 0x2d, 0xaa, 0x82, 0x94, 0xf9, 0x38, 0xf1, 0x58,
            0x9e, 0x4c, 0x3e, 0x00, 0x64, 0x67, 0xda, 0x9e, 0x8b, 0x5c, 0xb1, 0xaa, 0xa8, 0x85,
            0x43, 0xfd, 0x1a, 0xf2, 0xd8, 0xa7, 0xa7, 0x31, 0x55, 0x73, 0x91, 0x19, 0x5e, 0x43,
            0xe3, 0xc0, 0xfb, 0xd0, 0xc6, 0xa8, 0x72, 0x43, 0x33, 0x2f, 0x69, 0x5c, 0x64, 0x92,
            0xc7, 0x17, 0xb2, 0x30, 0x7a, 0xc1, 0x0a, 0x0d, 0x30, 0xbb, 0x94, 0xcb, 0x5c, 0x49,
            0x88, 0xe0, 0xb4, 0x0b, 0x4e, 0xab, 0xd7, 0x8e, 0x2d, 0x82, 0x55, 0x33, 0xb1, 0x00,
            0xa6, 0x89, 0x32, 0x59, 0x86, 0xde, 0xd7, 0x13, 0xea, 0x35, 0x0a, 0xa0, 0x50, 0x89,
            0x95, 0xe7, 0xaf, 0xaa, 0x6a, 0x4e, 0x22, 0xb4, 0x7f, 0x2e, 0x49, 0x9d, 0x67, 0x3a,
            0x95, 0x99, 0x75, 0x0a, 0x6b, 0x4d, 0x3e, 0x9d, 0x03, 0x1e, 0xfd, 0x82, 0xda, 0x02,
            0x3e, 0x18, 0xe4, 0x26, 0xdf, 0xb0, 0x1d, 0x49, 0xce, 0x6c, 0xf8, 0xbc, 0xbe, 0x82,
            0x27, 0x0e, 0x66, 0xa1, 0xc1, 0x85, 0xe2, 0xe1, 0x03, 0x83, 0xa4, 0x82, 0xf7, 0xd0,
            0x66, 0x12, 0x8b, 0xc4,
        ];
        let expected = &buf[1..];

        let actual: Vec<u8> = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);

        let actual: &[u8] = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);
    }

    #[test]
    fn tagged_enums() {
        #[derive(Serialize, Deserialize)]
        enum TestEnum<'a> {
            #[serde(rename = "19")]
            Unit,
            #[serde(rename = "235")]
            NewType(u64),
            #[serde(rename = "179")]
            Tuple(u32, u64, Vec<u16>),
            #[serde(rename = "97")]
            Struct {
                #[serde(borrow, with = "serde_bytes")]
                data: Cow<'a, [u8]>,
                footer: u32,
            },
        }

        assert!(matches!(from_slice(&[19]).unwrap(), TestEnum::Unit));

        let buf = [235, 0xa7, 0xc5, 0x31, 0x9c, 0x8d, 0x87, 0x48, 0xd2];
        if let TestEnum::NewType(v) = from_slice(&buf).unwrap() {
            assert_eq!(v, 0xa7c5319c8d8748d2);
        } else {
            panic!();
        }

        let buf = [
            179, 0x60, 0xfb, 0x4d, 0x0d, 0xc4, 0x98, 0x40, 0x65, 0xf5, 0xdb, 0xbf, 0x3c, 0x05,
            0xa9, 0xca, 0xb9, 0xe7, 0x96, 0x3b, 0x74, 0xfa, 0x82, 0xb2,
        ];
        if let TestEnum::Tuple(a, b, c) = from_slice(&buf).unwrap() {
            assert_eq!(a, 0x60fb4d0d);
            assert_eq!(b, 0xc4984065f5dbbf3c);
            assert_eq!(c, &[0xa9ca, 0xb9e7, 0x963b, 0x74fa, 0x82b2]);
        } else {
            panic!();
        }

        let buf = [
            97, 0x0b, 0xc2, 0xfd, 0xd6, 0xa1, 0xed, 0x8a, 0x12, 0x46, 0xd4, 0x20, 0xaf, 0xcc, 0x88,
            0x8c, 0xd2,
        ];
        if let TestEnum::Struct { data, footer } = from_slice(&buf).unwrap() {
            assert_eq!(
                &*data,
                &[0xc2, 0xfd, 0xd6, 0xa1, 0xed, 0x8a, 0x12, 0x46, 0xd4, 0x20, 0xaf]
            );
            assert_eq!(footer, 0xcc888cd2);
        } else {
            panic!();
        }
    }

    #[test]
    fn unknown_enum_variant() {
        #[derive(Debug, Serialize, Deserialize)]
        enum Unknown {
            #[serde(rename = "7")]
            Foo,
        }

        from_slice::<Unknown>(&[1]).expect_err("Deserialized unknown enum variant");
    }

    #[test]
    fn complex_struct() {
        #[derive(Debug, Eq, PartialEq, Serialize, Deserialize)]
        struct Address<'a> {
            #[serde(borrow, with = "serde_bytes")]
            bytes: Cow<'a, [u8]>,
        }

        #[derive(Debug, Eq, PartialEq, Serialize, Deserialize)]
        struct Info<'a> {
            #[serde(borrow)]
            addrs: Vec<Address<'a>>,
            expiration: u64,
        }

        #[derive(Debug, Eq, PartialEq, Serialize, Deserialize)]
        struct Upgrade<'a> {
            index: u32,
            #[serde(borrow)]
            info: Info<'a>,
        }

        let expected = Upgrade {
            index: 7,
            info: Info {
                addrs: vec![
                    Address {
                        bytes: Cow::Owned(vec![
                            0x4f, 0x58, 0x50, 0x9e, 0xb6, 0x8b, 0x9d, 0x19, 0x9e, 0x00, 0x92, 0x5e,
                            0xcb, 0x0f, 0xfd, 0x53, 0x80, 0x06, 0xfe, 0xc3,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0xb6, 0x7c, 0xd5, 0xef, 0x88, 0x00, 0xa7, 0xbc, 0xba, 0x2e, 0xfb, 0x91,
                            0x09, 0x33, 0xee, 0x51, 0xdd, 0x02, 0x24, 0x35,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x2b, 0x05, 0x87, 0x83, 0x8a, 0x2a, 0xe9, 0xc4, 0x0e, 0x54, 0x28, 0x11,
                            0xc2, 0x99, 0x33, 0xa8, 0x65, 0xd4, 0x6c, 0x3d,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x08, 0xd2, 0xb5, 0x03, 0x64, 0xb5, 0x27, 0x7f, 0xf0, 0xaf, 0x90, 0x6d,
                            0x03, 0x10, 0x21, 0xb3, 0x20, 0xdd, 0xfb, 0xda,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0xec, 0xc9, 0x7d, 0x9d, 0x6c, 0x68, 0x4e, 0x43, 0x6e, 0x39, 0x51, 0xe0,
                            0xa8, 0x6f, 0x49, 0xf1, 0xf4, 0xd3, 0xdb, 0x2a,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x11, 0xed, 0x25, 0xe6, 0x6b, 0xed, 0x56, 0x25, 0x87, 0xb4, 0x1c, 0x94,
                            0x9c, 0x81, 0xcf, 0x2c, 0x34, 0xb8, 0x5e, 0xc3,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x3d, 0x82, 0xcb, 0x29, 0xe8, 0xa7, 0x34, 0x37, 0x3a, 0x46, 0x07, 0xa4,
                            0xf2, 0xb3, 0x94, 0xb0, 0x73, 0xed, 0x86, 0x3b,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x99, 0xa4, 0xb5, 0x89, 0x01, 0x59, 0x18, 0x01, 0x08, 0x53, 0xcf, 0x17,
                            0x21, 0x14, 0x65, 0xcf, 0x05, 0x7f, 0xaa, 0x5d,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0xcc, 0x38, 0x3b, 0x85, 0xde, 0xc2, 0x59, 0xe6, 0x22, 0xee, 0xa4, 0xea,
                            0x83, 0x72, 0x08, 0x7e, 0xdf, 0xea, 0xe1, 0xc3,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x7a, 0xd9, 0x4d, 0x53, 0x9c, 0xc2, 0xff, 0xe3, 0x1d, 0xd6, 0x60, 0x78,
                            0x31, 0xb3, 0x2f, 0x76, 0x12, 0xb7, 0xc7, 0xaf,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x10, 0x88, 0xf6, 0x6f, 0x1d, 0x27, 0x2d, 0xad, 0x5b, 0x48, 0xca, 0xaf,
                            0xba, 0x63, 0x99, 0xbe, 0x23, 0x3b, 0xd5, 0xca,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0x49, 0x91, 0xa9, 0x0f, 0x47, 0xcd, 0xfe, 0xdb, 0xd6, 0xfb, 0xb3, 0xe9,
                            0xa4, 0xc2, 0xc2, 0x15, 0xb3, 0xe7, 0xe5, 0xb6,
                        ]),
                    },
                    Address {
                        bytes: Cow::Owned(vec![
                            0xdd, 0xe3, 0x77, 0xb0, 0xc3, 0x1b, 0x56, 0x2c, 0x90, 0x67, 0x88, 0xc6,
                            0xc5, 0xa5, 0xd8, 0xb8, 0xee, 0xc3, 0xa0, 0x87,
                        ]),
                    },
                ],
                expiration: 0x90e4_9c5d_cb20_0792,
            },
        };
        let buf = [
            0x00, 0x00, 0x00, 0x07, 0x0d, 0x14, 0x4f, 0x58, 0x50, 0x9e, 0xb6, 0x8b, 0x9d, 0x19,
            0x9e, 0x00, 0x92, 0x5e, 0xcb, 0x0f, 0xfd, 0x53, 0x80, 0x06, 0xfe, 0xc3, 0x14, 0xb6,
            0x7c, 0xd5, 0xef, 0x88, 0x00, 0xa7, 0xbc, 0xba, 0x2e, 0xfb, 0x91, 0x09, 0x33, 0xee,
            0x51, 0xdd, 0x02, 0x24, 0x35, 0x14, 0x2b, 0x05, 0x87, 0x83, 0x8a, 0x2a, 0xe9, 0xc4,
            0x0e, 0x54, 0x28, 0x11, 0xc2, 0x99, 0x33, 0xa8, 0x65, 0xd4, 0x6c, 0x3d, 0x14, 0x08,
            0xd2, 0xb5, 0x03, 0x64, 0xb5, 0x27, 0x7f, 0xf0, 0xaf, 0x90, 0x6d, 0x03, 0x10, 0x21,
            0xb3, 0x20, 0xdd, 0xfb, 0xda, 0x14, 0xec, 0xc9, 0x7d, 0x9d, 0x6c, 0x68, 0x4e, 0x43,
            0x6e, 0x39, 0x51, 0xe0, 0xa8, 0x6f, 0x49, 0xf1, 0xf4, 0xd3, 0xdb, 0x2a, 0x14, 0x11,
            0xed, 0x25, 0xe6, 0x6b, 0xed, 0x56, 0x25, 0x87, 0xb4, 0x1c, 0x94, 0x9c, 0x81, 0xcf,
            0x2c, 0x34, 0xb8, 0x5e, 0xc3, 0x14, 0x3d, 0x82, 0xcb, 0x29, 0xe8, 0xa7, 0x34, 0x37,
            0x3a, 0x46, 0x07, 0xa4, 0xf2, 0xb3, 0x94, 0xb0, 0x73, 0xed, 0x86, 0x3b, 0x14, 0x99,
            0xa4, 0xb5, 0x89, 0x01, 0x59, 0x18, 0x01, 0x08, 0x53, 0xcf, 0x17, 0x21, 0x14, 0x65,
            0xcf, 0x05, 0x7f, 0xaa, 0x5d, 0x14, 0xcc, 0x38, 0x3b, 0x85, 0xde, 0xc2, 0x59, 0xe6,
            0x22, 0xee, 0xa4, 0xea, 0x83, 0x72, 0x08, 0x7e, 0xdf, 0xea, 0xe1, 0xc3, 0x14, 0x7a,
            0xd9, 0x4d, 0x53, 0x9c, 0xc2, 0xff, 0xe3, 0x1d, 0xd6, 0x60, 0x78, 0x31, 0xb3, 0x2f,
            0x76, 0x12, 0xb7, 0xc7, 0xaf, 0x14, 0x10, 0x88, 0xf6, 0x6f, 0x1d, 0x27, 0x2d, 0xad,
            0x5b, 0x48, 0xca, 0xaf, 0xba, 0x63, 0x99, 0xbe, 0x23, 0x3b, 0xd5, 0xca, 0x14, 0x49,
            0x91, 0xa9, 0x0f, 0x47, 0xcd, 0xfe, 0xdb, 0xd6, 0xfb, 0xb3, 0xe9, 0xa4, 0xc2, 0xc2,
            0x15, 0xb3, 0xe7, 0xe5, 0xb6, 0x14, 0xdd, 0xe3, 0x77, 0xb0, 0xc3, 0x1b, 0x56, 0x2c,
            0x90, 0x67, 0x88, 0xc6, 0xc5, 0xa5, 0xd8, 0xb8, 0xee, 0xc3, 0xa0, 0x87, 0x90, 0xe4,
            0x9c, 0x5d, 0xcb, 0x20, 0x07, 0x92,
        ];

        let actual = from_slice(&buf).unwrap();
        assert_eq!(expected, actual);
    }
}
