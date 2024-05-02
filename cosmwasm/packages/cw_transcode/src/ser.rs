use std::fmt::Display;

use cosmwasm_std::{Attribute, Event};
use serde::{
    ser::{Impossible, SerializeStruct, SerializeStructVariant},
    Serialize,
};

use crate::Error;

pub struct Serializer(Option<Event>);

impl Serializer {
    pub const fn new() -> Self {
        Self(None)
    }

    pub fn finish(self) -> Option<Event> {
        self.0
    }
}

impl Default for Serializer {
    fn default() -> Self {
        Self::new()
    }
}

impl<'a> serde::Serializer for &'a mut Serializer {
    type Ok = ();
    type Error = Error;

    type SerializeSeq = Impossible<Self::Ok, Self::Error>;
    type SerializeTuple = Impossible<Self::Ok, Self::Error>;
    type SerializeTupleStruct = Impossible<Self::Ok, Self::Error>;
    type SerializeTupleVariant = Impossible<Self::Ok, Self::Error>;
    type SerializeMap = Impossible<Self::Ok, Self::Error>;
    type SerializeStruct = Transcoder<'a>;
    type SerializeStructVariant = Transcoder<'a>;

    #[inline]
    fn serialize_bool(self, _: bool) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_i8(self, _: i8) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_i16(self, _: i16) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_i32(self, _: i32) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_i64(self, _: i64) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_i128(self, _: i128) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_u8(self, _: u8) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_u16(self, _: u16) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_u32(self, _: u32) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_u64(self, _: u64) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_u128(self, _: u128) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_f32(self, _: f32) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_f64(self, _: f64) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_char(self, _: char) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_str(self, _: &str) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    fn serialize_bytes(self, _: &[u8]) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_none(self) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_some<T>(self, value: &T) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Serialize,
    {
        value.serialize(self)
    }

    #[inline]
    fn serialize_unit(self) -> Result<Self::Ok, Self::Error> {
        Err(Error::NotAStruct)
    }

    fn serialize_unit_struct(self, name: &'static str) -> Result<Self::Ok, Self::Error> {
        if self.0.is_none() {
            self.0 = Some(Event::new(name));
            Ok(())
        } else {
            Err(Error::MultipleStructs)
        }
    }

    fn serialize_unit_variant(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
    ) -> Result<Self::Ok, Self::Error> {
        if self.0.is_none() {
            self.0 = Some(Event::new(format!("{name}::{variant}")));
            Ok(())
        } else {
            Err(Error::MultipleStructs)
        }
    }

    #[inline]
    fn serialize_newtype_struct<T>(self, _: &'static str, _: &T) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Serialize,
    {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_newtype_variant<T>(
        self,
        _: &'static str,
        _variant_index: u32,
        _: &'static str,
        _: &T,
    ) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Serialize,
    {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_tuple_variant(
        self,
        _: &'static str,
        _variant_index: u32,
        _: &'static str,
        _: usize,
    ) -> Result<Self::SerializeTupleVariant, Self::Error> {
        Err(Error::NotAStruct)
    }

    fn serialize_struct_variant(
        self,
        name: &'static str,
        _variant_index: u32,
        variant: &'static str,
        _: usize,
    ) -> Result<Self::SerializeStructVariant, Self::Error> {
        if self.0.is_none() {
            Ok(Transcoder {
                serializer: self,
                event: Event::new(format!("{name}::{variant}")),
            })
        } else {
            Err(Error::MultipleStructs)
        }
    }

    #[inline]
    fn serialize_seq(self, _: Option<usize>) -> Result<Self::SerializeSeq, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_tuple(self, _: usize) -> Result<Self::SerializeTuple, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_tuple_struct(
        self,
        _: &'static str,
        _: usize,
    ) -> Result<Self::SerializeTupleStruct, Self::Error> {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn serialize_map(self, _: Option<usize>) -> Result<Self::SerializeMap, Self::Error> {
        Err(Error::NotAStruct)
    }

    fn serialize_struct(
        self,
        name: &'static str,
        _: usize,
    ) -> Result<Self::SerializeStruct, Self::Error> {
        if self.0.is_none() {
            Ok(Transcoder {
                serializer: self,
                event: Event::new(name),
            })
        } else {
            Err(Error::MultipleStructs)
        }
    }

    #[inline]
    fn collect_str<T>(self, _: &T) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Display,
    {
        Err(Error::NotAStruct)
    }

    #[inline]
    fn is_human_readable(&self) -> bool {
        true
    }
}

pub struct Transcoder<'a> {
    serializer: &'a mut Serializer,
    event: Event,
}

impl<'a> Transcoder<'a> {
    fn serialize_field<T>(&mut self, k: &'static str, v: &T) -> Result<(), Error>
    where
        T: ?Sized + Serialize,
    {
        let key = k.into();
        let value = serde_json_wasm::to_string(v)?;
        self.event.attributes.push(Attribute { key, value });

        Ok(())
    }

    #[inline]
    fn end(self) -> Result<(), Error> {
        self.serializer.0 = Some(self.event);
        Ok(())
    }
}

impl<'a> SerializeStruct for Transcoder<'a> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T>(&mut self, k: &'static str, v: &T) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Serialize,
    {
        self.serialize_field(k, v)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        self.end()
    }
}

impl<'a> SerializeStructVariant for Transcoder<'a> {
    type Ok = ();
    type Error = Error;

    #[inline]
    fn serialize_field<T>(&mut self, k: &'static str, v: &T) -> Result<Self::Ok, Self::Error>
    where
        T: ?Sized + Serialize,
    {
        self.serialize_field(k, v)
    }

    #[inline]
    fn end(self) -> Result<Self::Ok, Self::Error> {
        self.end()
    }
}
