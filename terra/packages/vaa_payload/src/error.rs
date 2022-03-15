use std::{
    fmt::Display,
    io,
    num::ParseIntError,
};

use serde::{
    de,
    ser,
};
use thiserror::Error as ThisError;

#[derive(Debug, ThisError)]
pub enum Error {
    #[error("{0}")]
    Message(Box<str>),
    #[error("{0}")]
    Io(#[from] io::Error),
    #[error("unexpected end of input")]
    Eof,
    #[error("`deserialize_any` is not supported")]
    DeserializeAnyNotSupported,
    #[error("trailing data in input buffer")]
    TrailingData,
    #[error("this type is not supported")]
    Unsupported,
    #[error("data is too large ({0} bytes), max supported length = 255")]
    DataTooLarge(usize),
    #[error("enum variant {0}::{1} cannot be parsed as a `u8`: {2}")]
    EnumVariant(&'static str, &'static str, ParseIntError),
    #[error("sequence length must be known befor serialization")]
    UnknownSequenceLength,
}

impl de::Error for Error {
    fn custom<T: Display>(msg: T) -> Error {
        Error::Message(msg.to_string().into_boxed_str())
    }
}

impl ser::Error for Error {
    fn custom<T: Display>(msg: T) -> Error {
        Error::Message(msg.to_string().into_boxed_str())
    }
}
