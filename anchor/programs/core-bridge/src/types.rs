use std::fmt;

use crate::{
    error::CoreBridgeError,
    message::{WormDecode, WormEncode},
};
use anchor_lang::prelude::*;
use solana_program::keccak;

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum VaaVersion {
    #[default]
    Unset,
    V1,
}

impl WormDecode for VaaVersion {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        match u8::decode_reader(reader)? {
            0 => Ok(VaaVersion::Unset),
            1 => Ok(VaaVersion::V1),
            _ => Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "invalid vaa version",
            )),
        }
    }
}

impl WormEncode for VaaVersion {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        match self {
            VaaVersion::Unset => 0u8.encode(writer),
            VaaVersion::V1 => 1u8.encode(writer),
        }
    }
}

#[derive(Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub enum Commitment {
    Confirmed,
    Finalized,
}

impl From<Commitment> for Finality {
    fn from(value: Commitment) -> Self {
        match value {
            Commitment::Confirmed => Finality { value: 1 },
            Commitment::Finalized => Finality { value: 32 },
        }
    }
}

/// Struct representing finality (consistency level). The underlying (u8) value is determined by
/// whichever network a Wormhole message is emitted from. For Solana, it is a function of number
/// of confirmations.
///
/// NOTE: It is preferred that this struct be a tuple struct, but Anchor's IDL generation cannot
/// handle non-named structs yet.
#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub struct Finality {
    value: u8,
}

impl From<u8> for Finality {
    fn from(value: u8) -> Finality {
        Finality { value }
    }
}

impl From<Finality> for u8 {
    fn from(finality: Finality) -> u8 {
        finality.value
    }
}

impl From<&Finality> for u8 {
    fn from(finality: &Finality) -> u8 {
        finality.value
    }
}

impl WormDecode for Finality {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let value = u8::decode_reader(reader)?;
        Ok(Self { value })
    }
}

impl WormEncode for Finality {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.value.encode(writer)
    }
}

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub struct ChainId {
    value: u16,
}

impl ChainId {
    pub fn to_be_bytes(self) -> [u8; 2] {
        self.value.to_be_bytes()
    }

    pub fn from_be_bytes(bytes: [u8; 2]) -> ChainId {
        u16::from_be_bytes(bytes).into()
    }
}

impl From<u16> for ChainId {
    fn from(value: u16) -> ChainId {
        ChainId { value }
    }
}

impl From<ChainId> for u16 {
    fn from(chain_id: ChainId) -> u16 {
        chain_id.value
    }
}

impl From<&ChainId> for u16 {
    fn from(chain_id: &ChainId) -> u16 {
        chain_id.value
    }
}

impl WormDecode for ChainId {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        u16::decode_reader(reader).map(Into::into)
    }
}

impl WormEncode for ChainId {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        u16::from(self).encode(writer)
    }
}

impl fmt::Display for ChainId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}

#[derive(Copy, Debug, Clone)]
pub struct SolanaChain;

impl TryFrom<ChainId> for SolanaChain {
    type Error = anchor_lang::error::Error;

    fn try_from(chain_id: ChainId) -> std::result::Result<SolanaChain, Self::Error> {
        match chain_id.value {
            1 => Ok(SolanaChain),
            _ => err!(CoreBridgeError::InvalidChain),
        }
    }
}

impl From<SolanaChain> for ChainId {
    fn from(_: SolanaChain) -> ChainId {
        ChainId { value: 1 }
    }
}

impl From<&SolanaChain> for ChainId {
    fn from(chain: &SolanaChain) -> ChainId {
        ChainId::from(*chain)
    }
}

impl WormDecode for SolanaChain {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let chain_id = ChainId::decode_reader(reader)?;
        match SolanaChain::try_from(chain_id) {
            Ok(chain) => Ok(chain),
            Err(_) => Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "invalid chain id",
            )),
        }
    }
}

impl WormEncode for SolanaChain {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        ChainId::from(self).encode(writer)
    }
}

/// Struct representing 32-byte addresses. These addresses are used to represent emitters, arbitrary
/// senders and recipients.
///
/// NOTE: It is preferred that this struct be a tuple struct, but Anchor's IDL generation cannot
/// handle non-named structs yet.
#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub struct ExternalAddress {
    bytes: [u8; 32],
}

impl AsRef<[u8]> for ExternalAddress {
    fn as_ref(&self) -> &[u8] {
        &self.bytes
    }
}

impl From<[u8; 32]> for ExternalAddress {
    fn from(bytes: [u8; 32]) -> Self {
        Self { bytes }
    }
}

impl From<ExternalAddress> for [u8; 32] {
    fn from(address: ExternalAddress) -> Self {
        address.bytes
    }
}

impl From<Pubkey> for ExternalAddress {
    fn from(pubkey: Pubkey) -> Self {
        Self {
            bytes: pubkey.to_bytes(),
        }
    }
}

impl From<ExternalAddress> for Pubkey {
    fn from(value: ExternalAddress) -> Self {
        Pubkey::from(value.bytes)
    }
}

impl WormDecode for ExternalAddress {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let mut bytes = [0; 32];
        reader.read_exact(&mut bytes)?;
        Ok(Self { bytes })
    }
}

impl WormEncode for ExternalAddress {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.bytes.encode(writer)
    }
}

impl fmt::Display for ExternalAddress {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "0x{}", hex::encode(self.bytes))
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

impl WormEncode for Timestamp {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.value.encode(writer)
    }
}

impl WormDecode for Timestamp {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let value = u32::decode_reader(reader)?;
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

impl WormEncode for Duration {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        self.seconds.encode(writer)
    }
}

impl WormDecode for Duration {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let seconds = u32::decode_reader(reader)?;
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
