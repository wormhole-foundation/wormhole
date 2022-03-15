use std::{convert::TryFrom, fmt::Display, io::Write};

use serde::{ser, Serialize};

use crate::Error;

/// A struct that serializes a rust value into the VAA payload wire format.
pub struct Serializer<W> {
    writer: W,
}

impl<W: Write> Serializer<W> {
    pub fn new(writer: W) -> Self {
        Self { writer }
    }
}

impl<'a, W: Write> ser::Serializer for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    type SerializeSeq = Self;
    type SerializeTuple = Self;
    type SerializeTupleStruct = Self;
    type SerializeTupleVariant = Self;
    type SerializeMap = Self;
    type SerializeStruct = Self;
    type SerializeStructVariant = Self;

    #[inline]
    fn serialize_bool(self, v: bool) -> Result<Self::Ok, Self::Error> {
        if v {
            self.writer.write_all(&[1]).map_err(Error::from)
        } else {
            self.writer.write_all(&[0]).map_err(Error::from)
        }
    }

    #[inline]
    fn serialize_i8(self, v: i8) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_i16(self, v: i16) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_i32(self, v: i32) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_i64(self, v: i64) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_i128(self, v: i128) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_u8(self, v: u8) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_u16(self, v: u16) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_u32(self, v: u32) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_u64(self, v: u64) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_u128(self, v: u128) -> Result<Self::Ok, Self::Error> {
        self.writer.write_all(&v.to_be_bytes()).map_err(Error::from)
    }

    #[inline]
    fn serialize_f32(self, _v: f32) -> Result<Self::Ok, Self::Error> {
        Err(Error::Unsupported)
    }

    #[inline]
    fn serialize_f64(self, _v: f64) -> Result<Self::Ok, Self::Error> {
        Err(Error::Unsupported)
    }

    #[inline]
    fn serialize_char(self, v: char) -> Result<Self::Ok, Self::Error> {
        self.serialize_u32(v.into())
    }

    #[inline]
    fn serialize_str(self, v: &str) -> Result<Self::Ok, Self::Error> {
        self.serialize_bytes(v.as_bytes())
    }

    fn serialize_bytes(self, v: &[u8]) -> Result<Self::Ok, Self::Error> {
        let len = u8::try_from(v.len()).map_err(|_| Error::SequenceTooLarge(v.len()))?;

        self.writer.write_all(&[len])?;
        self.writer.write_all(v).map_err(Error::from)
    }

    #[inline]
    fn serialize_none(self) -> Result<Self::Ok, Self::Error> {
        Err(Error::Unsupported)
    }

    #[inline]
    fn serialize_some<T: ?Sized>(self, value: &T) -> Result<Self::Ok, Self::Error>
    where
        T: Serialize,
    {
        value.serialize(self)
    }

    #[inline]
    fn serialize_unit(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }

    #[inline]
    fn serialize_unit_struct(self, _name: &'static str) -> Result<Self::Ok, Self::Error> {
        self.serialize_unit()
    }

    fn serialize_unit_variant(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
    ) -> Result<Self::Ok, Self::Error> {
        let v: u8 = variant
            .parse()
            .map_err(|e| Error::EnumVariant(name, variant, e))?;

        self.writer.write_all(&[v]).map_err(Error::from)
    }

    #[inline]
    fn serialize_newtype_struct<T: ?Sized>(
        self,
        _name: &'static str,
        value: &T,
    ) -> Result<Self::Ok, Self::Error>
    where
        T: Serialize,
    {
        value.serialize(self)
    }

    fn serialize_newtype_variant<T: ?Sized>(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
        value: &T,
    ) -> Result<Self::Ok, Self::Error>
    where
        T: Serialize,
    {
        let v: u8 = variant
            .parse()
            .map_err(|e| Error::EnumVariant(name, variant, e))?;

        self.writer.write_all(&[v])?;
        value.serialize(self)
    }

    fn serialize_tuple_variant(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
        _len: usize,
    ) -> Result<Self::SerializeTupleVariant, Self::Error> {
        let v: u8 = variant
            .parse()
            .map_err(|e| Error::EnumVariant(name, variant, e))?;

        self.writer.write_all(&[v])?;
        Ok(self)
    }

    fn serialize_struct_variant(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
        _len: usize,
    ) -> Result<Self::SerializeStructVariant, Self::Error> {
        let v: u8 = variant
            .parse()
            .map_err(|e| Error::EnumVariant(name, variant, e))?;

        self.writer.write_all(&[v])?;
        Ok(self)
    }

    fn serialize_seq(self, len: Option<usize>) -> Result<Self::SerializeSeq, Self::Error> {
        let len = len
            .ok_or(Error::UnknownSequenceLength)
            .and_then(|v| u8::try_from(v).map_err(|_| Error::SequenceTooLarge(v)))?;

        self.writer.write_all(&[len])?;
        Ok(self)
    }

    #[inline]
    fn serialize_tuple(self, _len: usize) -> Result<Self::SerializeTuple, Self::Error> {
        Ok(self)
    }

    #[inline]
    fn serialize_tuple_struct(
        self,
        _name: &'static str,
        _len: usize,
    ) -> Result<Self::SerializeTupleStruct, Self::Error> {
        Ok(self)
    }

    #[inline]
    fn serialize_map(self, len: Option<usize>) -> Result<Self::SerializeMap, Self::Error> {
        let len = len
            .ok_or(Error::UnknownSequenceLength)
            .and_then(|v| u8::try_from(v).map_err(|_| Error::SequenceTooLarge(v)))?;

        self.writer.write_all(&[len])?;
        Ok(self)
    }

    #[inline]
    fn serialize_struct(
        self,
        _name: &'static str,
        _len: usize,
    ) -> Result<Self::SerializeStruct, Self::Error> {
        Ok(self)
    }

    #[inline]
    fn collect_str<T: ?Sized>(self, value: &T) -> Result<Self::Ok, Self::Error>
    where
        T: Display,
    {
        self.serialize_str(&value.to_string())
    }

    #[inline]
    fn is_human_readable(&self) -> bool {
        false
    }
}

impl<'a, W: Write> ser::SerializeSeq for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_element<T: ?Sized>(&mut self, value: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeTuple for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_element<T: ?Sized>(&mut self, value: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeTupleStruct for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T: ?Sized>(&mut self, value: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeTupleVariant for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T: ?Sized>(&mut self, value: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeStruct for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T: ?Sized>(
        &mut self,
        _key: &'static str,
        value: &T,
    ) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeStructVariant for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T: ?Sized>(
        &mut self,
        _key: &'static str,
        value: &T,
    ) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

impl<'a, W: Write> ser::SerializeMap for &'a mut Serializer<W> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_key<T: ?Sized>(&mut self, key: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        key.serialize(&mut **self)
    }

    #[inline]
    fn serialize_value<T: ?Sized>(&mut self, value: &T) -> Result<(), Self::Error>
    where
        T: Serialize,
    {
        value.serialize(&mut **self)
    }

    fn serialize_entry<K: ?Sized, V: ?Sized>(
        &mut self,
        key: &K,
        value: &V,
    ) -> Result<(), Self::Error>
    where
        K: Serialize,
        V: Serialize,
    {
        self.serialize_key(key)?;
        self.serialize_value(value)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use std::{borrow::Cow, collections::BTreeMap};

    use serde::{Deserialize, Serialize};

    use crate::{to_vec, to_writer, Error};

    #[test]
    fn empty_buffer() {
        to_writer(&mut [][..], &0xcc1a_7e0e_31f4_eae4u64)
            .expect_err("serialized data to empty buffer");
    }

    #[test]
    fn bool() {
        assert_eq!(to_vec(&true).unwrap(), &[1]);
        assert_eq!(to_vec(&false).unwrap(), &[0]);
    }

    #[test]
    fn integers() {
        macro_rules! check {
            ($v:ident, $ty:ty) => {
                // Casting an integer from a larger width to a smaller width will truncate the
                // upper bits.
                let v = $v as $ty;
                let expected = v.to_be_bytes();

                let actual = to_vec(&v).expect("failed to serialize integer");
                assert_eq!(actual, &expected);
            };
            ($v:ident, $($ty:ty),*) => {
                $(
                    check!($v, $ty);
                )*
            };
        }

        // Value randomly generated from `dd if=/dev/urandom | xxd -p -l 16`.
        let v = 0x46b1_265e_2f09_2e98_15c5_4c28_5c53_986cu128;
        check!(v, i8, i16, i32, i64, i128, u8, u16, u32, u64, u128);
    }

    #[test]
    fn strings() {
        let buf = "Löwe 老虎 Léopard";

        let expected = IntoIterator::into_iter([buf.len() as u8])
            .chain(buf.as_bytes().into_iter().copied())
            .collect::<Vec<u8>>();
        let actual = to_vec(buf).unwrap();
        assert_eq!(expected, actual);

        let actual = to_vec(&buf.to_string()).unwrap();
        assert_eq!(expected, actual);

        let actual = to_vec(&"").unwrap();
        assert_eq!(&[0], &*actual);
    }

    #[test]
    fn maps() {
        let m = BTreeMap::from([
            (0xb74909e6u32, 0xe3a9db9cu32),
            (0x383c5309u32, 0x5c6b2d54u32),
        ]);

        let expected = [
            0x02, 0x38, 0x3c, 0x53, 0x09, 0x5c, 0x6b, 0x2d, 0x54, 0xb7, 0x49, 0x09, 0xe6, 0xe3,
            0xa9, 0xdb, 0x9c,
        ];
        let actual = to_vec(&m).unwrap();
        assert_eq!(actual, expected);

        let actual = to_vec(&BTreeMap::<u32, u32>::new()).unwrap();
        assert_eq!(actual, [0]);
    }

    #[test]
    fn empty_bytes() {
        let expected = [0x0];
        let actual = to_vec::<Vec<u16>>(&vec![]).unwrap();
        assert_eq!(actual, &expected);

        let actual = to_vec::<[u16]>(&[]).unwrap();
        assert_eq!(actual, &expected);
    }

    #[test]
    fn bytes() {
        let expected = [
            0x40, 0xff, 0x17, 0xc6, 0x15, 0xa4, 0x7c, 0x3c, 0x8c, 0xf3, 0x8b, 0x9f, 0x88, 0x31,
            0xc4, 0x46, 0x07, 0xcb, 0xe9, 0x2d, 0xc4, 0x59, 0xaf, 0x34, 0x5a, 0x32, 0x66, 0x8c,
            0x05, 0xc9, 0x3d, 0xab, 0x4f, 0xd2, 0x8a, 0xb6, 0x2e, 0x68, 0x58, 0x45, 0xef, 0x56,
            0x27, 0x1a, 0xb6, 0xe0, 0x17, 0x47, 0xf3, 0xb8, 0x5d, 0xbd, 0x1b, 0x92, 0xd9, 0xdd,
            0xe2, 0x99, 0x04, 0xbb, 0x67, 0xf7, 0x9f, 0xe1, 0xe1,
        ];

        let actual = to_vec(&expected[1..].to_vec()).unwrap();
        assert_eq!(actual, expected);

        let actual = to_vec(&expected[1..]).unwrap();
        assert_eq!(actual, expected);
    }

    #[test]
    fn max_bytes() {
        let expected = [
            0xff, 0x79, 0xce, 0xa2, 0x42, 0xd5, 0x8d, 0x0a, 0xaf, 0xa1, 0x72, 0x01, 0x92, 0xfc,
            0x23, 0x4f, 0x80, 0x56, 0x9d, 0xff, 0x9d, 0x44, 0x30, 0xe3, 0x22, 0xb6, 0xd5, 0x11,
            0xdd, 0xbe, 0x68, 0x6b, 0x34, 0x53, 0x5a, 0x97, 0x1b, 0x22, 0x96, 0x6c, 0xd5, 0xc6,
            0x08, 0x2a, 0xf0, 0x1b, 0x74, 0x22, 0xe8, 0xdf, 0xcd, 0xad, 0xa0, 0x75, 0x43, 0x84,
            0xf5, 0x43, 0x66, 0x38, 0x42, 0x66, 0xbb, 0xa1, 0x10, 0x54, 0x6a, 0x00, 0x4c, 0x9c,
            0x0d, 0x53, 0xed, 0x72, 0xc7, 0x6c, 0x9c, 0x86, 0x75, 0xbe, 0x7d, 0xf3, 0x54, 0x70,
            0x25, 0xda, 0x96, 0x9b, 0xc8, 0x6e, 0xc5, 0xc1, 0x56, 0xcf, 0x5a, 0x8d, 0xe1, 0x12,
            0x8d, 0xd7, 0x06, 0x33, 0xc5, 0x25, 0xf2, 0x31, 0xa2, 0x42, 0x3b, 0xc8, 0x30, 0xc9,
            0x1e, 0x51, 0xa5, 0x6a, 0x52, 0x0d, 0x6c, 0xbb, 0xc7, 0xde, 0x44, 0x8e, 0xe0, 0x80,
            0x00, 0xcf, 0x4b, 0xf1, 0x5e, 0xff, 0x68, 0x9d, 0xb5, 0x13, 0xad, 0x71, 0x6a, 0x94,
            0x0d, 0x68, 0x37, 0x7f, 0x68, 0x47, 0xf6, 0x03, 0xc5, 0x08, 0xf2, 0x47, 0x90, 0x7d,
            0x29, 0xd8, 0xeb, 0x7d, 0xc2, 0xbb, 0xaa, 0xea, 0x0b, 0x1a, 0x73, 0x44, 0xd1, 0x35,
            0x42, 0x79, 0xd8, 0x2b, 0x99, 0xbb, 0x75, 0xb7, 0xad, 0x54, 0xd3, 0xbb, 0x7b, 0xa3,
            0x4d, 0x3a, 0xea, 0x74, 0xbe, 0x82, 0x40, 0xac, 0x63, 0x6e, 0x03, 0x38, 0x3c, 0x57,
            0xa2, 0x02, 0x8b, 0x6c, 0xc9, 0x32, 0x9f, 0x6a, 0x35, 0x8f, 0x2d, 0x4e, 0x4d, 0xc6,
            0x2b, 0x51, 0x08, 0x02, 0x35, 0x03, 0x45, 0xa1, 0x13, 0x0a, 0xad, 0x3c, 0x53, 0x90,
            0x18, 0xe1, 0x89, 0xf2, 0xeb, 0xf1, 0x57, 0x2d, 0x32, 0xc1, 0x1a, 0x46, 0x8d, 0x72,
            0xe4, 0x39, 0xbb, 0x75, 0xda, 0x85, 0xec, 0x8d, 0x98, 0x31, 0xf2, 0xfb, 0x20, 0x9a,
            0x4e, 0x9c, 0xe6, 0x8c,
        ];

        let actual = to_vec(&expected[1..].to_vec()).unwrap();
        assert_eq!(actual, expected);

        let actual = to_vec(&expected[1..]).unwrap();
        assert_eq!(actual, expected);
    }

    #[test]
    fn data_too_large() {
        let e =
            to_vec(&vec![0u16; 300]).expect_err("serialized sequence with more than 255 entries");
        assert!(matches!(e, Error::SequenceTooLarge(300)));
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

        assert_eq!(to_vec(&TestEnum::Unit).unwrap(), &[19]);

        let expected = [235, 0xa7, 0xc5, 0x31, 0x9c, 0x8d, 0x87, 0x48, 0xd2];
        assert_eq!(
            to_vec(&TestEnum::NewType(0xa7c5319c8d8748d2)).unwrap(),
            &expected
        );

        let expected = [
            179, 0x60, 0xfb, 0x4d, 0x0d, 0xc4, 0x98, 0x40, 0x65, 0xf5, 0xdb, 0xbf, 0x3c, 0x05,
            0xa9, 0xca, 0xb9, 0xe7, 0x96, 0x3b, 0x74, 0xfa, 0x82, 0xb2,
        ];
        let value = TestEnum::Tuple(
            0x60fb4d0d,
            0xc4984065f5dbbf3c,
            vec![0xa9ca, 0xb9e7, 0x963b, 0x74fa, 0x82b2],
        );
        assert_eq!(to_vec(&value).unwrap(), &expected);

        let expected = [
            97, 0x0b, 0xc2, 0xfd, 0xd6, 0xa1, 0xed, 0x8a, 0x12, 0x46, 0xd4, 0x20, 0xaf, 0xcc, 0x88,
            0x8c, 0xd2,
        ];
        let value = TestEnum::Struct {
            data: Cow::Owned(vec![
                0xc2, 0xfd, 0xd6, 0xa1, 0xed, 0x8a, 0x12, 0x46, 0xd4, 0x20, 0xaf,
            ]),
            footer: 0xcc888cd2,
        };
        assert_eq!(to_vec(&value).unwrap(), &expected);
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

        let expected = [
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
        let value = Upgrade {
            index: 7,
            info: Info {
                addrs: vec![
                    Address {
                        bytes: Cow::Borrowed(&expected[6..26]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[27..47]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[48..68]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[69..89]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[90..110]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[111..131]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[132..152]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[153..173]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[174..194]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[195..215]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[216..236]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[237..257]),
                    },
                    Address {
                        bytes: Cow::Borrowed(&expected[258..278]),
                    },
                ],
                expiration: 0x90e4_9c5d_cb20_0792,
            },
        };

        let actual = to_vec(&value).unwrap();
        assert_eq!(actual, expected);
    }
}
