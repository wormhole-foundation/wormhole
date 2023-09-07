use std::fmt;

use anchor_lang::prelude::*;
use solana_program::keccak;
use wormhole_vaas::{Readable, Writeable};

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum VaaVersion {
    #[default]
    Unset,
    V1,
}

impl VaaVersion {
    const UNSET: u8 = 0;
    const V_1: u8 = 1;
}

impl Readable for VaaVersion {
    const SIZE: Option<usize> = Some(VaaVersion::INIT_SPACE);

    fn read<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        match u8::read(reader)? {
            VaaVersion::UNSET => Ok(VaaVersion::Unset),
            VaaVersion::V_1 => Ok(VaaVersion::V1),
            _ => Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "invalid vaa version",
            )),
        }
    }
}

impl Writeable for VaaVersion {
    fn write<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        match self {
            VaaVersion::Unset => VaaVersion::UNSET.write(writer),
            VaaVersion::V1 => VaaVersion::V_1.write(writer),
        }
    }

    fn written_size(&self) -> usize {
        VaaVersion::INIT_SPACE
    }
}

#[derive(Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub enum Commitment {
    Confirmed,
    Finalized,
}

impl From<Commitment> for u8 {
    fn from(value: Commitment) -> Self {
        match value {
            Commitment::Confirmed => 1,
            Commitment::Finalized => 32,
        }
    }
}

/// This struct defines unix timestamp as u32 (as opposed to more modern systems
/// that have adopted i64). Methods for this struct are meant to convert
/// Solana's clock type to this type assuming we are far from year 2038.
#[derive(
    Default,
    Copy,
    PartialEq,
    Eq,
    Debug,
    Clone,
    AnchorSerialize,
    AnchorDeserialize,
    PartialOrd,
    Ord,
    InitSpace,
)]
pub struct Timestamp {
    value: u32,
}

impl Timestamp {
    /// In case we encounter the year 2038 problem, this method will return the max value for u32.
    pub fn saturating_add(&self, duration: &Duration) -> Timestamp {
        Timestamp {
            value: self.value.saturating_add(duration.seconds),
        }
    }

    pub fn to_be_bytes(self) -> [u8; 4] {
        self.value.to_be_bytes()
    }
}

impl From<u32> for Timestamp {
    fn from(value: u32) -> Self {
        Timestamp { value }
    }
}

impl From<Clock> for Timestamp {
    fn from(clock: Clock) -> Self {
        Self {
            value: clock.unix_timestamp.try_into().expect("timestamp overflow"),
        }
    }
}

impl Writeable for Timestamp {
    fn write<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.value.write(writer)
    }

    fn written_size(&self) -> usize {
        Timestamp::INIT_SPACE
    }
}

impl Readable for Timestamp {
    const SIZE: Option<usize> = Some(Timestamp::INIT_SPACE);

    fn read<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let value = u32::read(reader)?;
        Ok(Self { value })
    }
}

#[derive(
    Debug,
    Copy,
    AnchorSerialize,
    AnchorDeserialize,
    Clone,
    PartialEq,
    Eq,
    InitSpace,
    PartialOrd,
    Ord,
)]
pub struct Duration {
    seconds: u32,
}

impl From<u32> for Duration {
    fn from(seconds: u32) -> Self {
        Duration { seconds }
    }
}
impl From<&u32> for Duration {
    fn from(seconds: &u32) -> Self {
        Duration { seconds: *seconds }
    }
}

impl Writeable for Duration {
    fn write<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.seconds.write(writer)
    }

    fn written_size(&self) -> usize {
        Duration::INIT_SPACE
    }
}

impl Readable for Duration {
    const SIZE: Option<usize> = Some(Duration::INIT_SPACE);

    fn read<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let seconds = u32::read(reader)?;
        Ok(Self { seconds })
    }
}

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub struct MessageHash {
    bytes: [u8; 32],
}

impl AsRef<[u8]> for MessageHash {
    fn as_ref(&self) -> &[u8] {
        &self.bytes
    }
}

impl From<[u8; 32]> for MessageHash {
    fn from(bytes: [u8; 32]) -> Self {
        Self { bytes }
    }
}

impl From<keccak::Hash> for MessageHash {
    fn from(hash: keccak::Hash) -> Self {
        Self { bytes: hash.0 }
    }
}

impl From<MessageHash> for keccak::Hash {
    fn from(hash: MessageHash) -> Self {
        keccak::Hash::new_from_array(hash.bytes)
    }
}

impl fmt::Display for MessageHash {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self.bytes))
    }
}
