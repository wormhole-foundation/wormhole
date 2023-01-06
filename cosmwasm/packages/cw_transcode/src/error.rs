use std::fmt;

use serde::ser;

#[derive(Debug, thiserror::Error)]
pub enum Error {
    #[error("{0}")]
    Custom(Box<str>),
    #[error("cannot transcode non-struct type to `Event`")]
    NotAStruct,
    #[error("{0}")]
    Json(#[from] serde_json_wasm::ser::Error),
    #[error("cannot transcode more than one struct to `Event`")]
    MultipleStructs,
    #[error("no event produced by serialization")]
    NoEvent,
}

impl ser::Error for Error {
    fn custom<T>(msg: T) -> Self
    where
        T: fmt::Display,
    {
        Error::Custom(msg.to_string().into_boxed_str())
    }
}
