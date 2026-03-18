//! Provide Types and Data about Wormhole's supported chains.
//!
//! The Chain enum and its trait implementations are auto-generated from the Go SDK
//! source of truth at compile time by build.rs.

use std::{fmt, str::FromStr};

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use thiserror::Error;

// Include the generated code from build.rs
// This includes: Chain enum, From<u16>, From<Chain> for u16, Display, and FromStr
include!(concat!(env!("OUT_DIR"), "/chains_generated.rs"));

#[derive(Debug, Error)]
#[error("invalid chain: {0}")]
pub struct InvalidChainError(String);

// Manually implemented traits that aren't generated

impl Serialize for Chain {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u16((*self).into())
    }
}

impl<'de> Deserialize<'de> for Chain {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        <u16 as Deserialize>::deserialize(deserializer).map(Self::from)
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn isomorphic_from() {
        for i in 0u16..=u16::MAX {
            assert_eq!(i, u16::from(Chain::from(i)));
        }
    }

    #[test]
    fn isomorphic_display() {
        for i in 0u16..=u16::MAX {
            let c = Chain::from(i);
            assert_eq!(c, c.to_string().parse().unwrap());
        }
    }
}
