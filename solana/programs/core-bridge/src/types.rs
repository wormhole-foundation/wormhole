//! Various types used by the Core Bridge Program.

use std::fmt;

use anchor_lang::prelude::*;
use solana_program::keccak;
use wormhole_io::{Readable, Writeable};

/// Representation of Solana's commitment levels. This enum is not exhaustive because Wormhole only
/// considers these two commitment levels in its Guardian observation.
///
/// See <https://docs.solana.com/cluster/commitments> for more info.
#[derive(Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub enum Commitment {
    /// One confirmation.
    Confirmed,
    /// 32 confirmations.
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

/// This struct defines unix timestamp as u32 (as opposed to more modern systems that have adopted
/// i64). Methods for this struct are meant to convert Solana's clock type to this type assuming we
/// are far from year 2038.
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
    /// Return the memory representation of this underlying u32 value as a byte array in big-endian
    /// byte order.
    pub fn to_be_bytes(self) -> [u8; 4] {
        self.value.to_be_bytes()
    }
}

impl PartialEq<u32> for Timestamp {
    fn eq(&self, other: &u32) -> bool {
        self.value == *other
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

impl core::ops::Add<Duration> for Timestamp {
    type Output = Self;

    fn add(self, rhs: Duration) -> Self::Output {
        Self {
            value: self.value + rhs.seconds,
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

/// To be used with the [Timestamp] type, this struct defines a duration in seconds.
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

/// This type is used to represent a message hash (keccak).
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

/// This type is kind of silly. But because [PostedMessageV1](crate::state::PostedMessageV1) has the
/// emitter chain ID as a field, which is unnecessary since it is always Solana's chain ID, we use
/// this type to guarantee that the encoded chain ID is always `1`.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct ChainIdSolanaOnly {
    chain_id: u16,
}

impl Default for ChainIdSolanaOnly {
    fn default() -> Self {
        Self {
            chain_id: crate::constants::SOLANA_CHAIN,
        }
    }
}
