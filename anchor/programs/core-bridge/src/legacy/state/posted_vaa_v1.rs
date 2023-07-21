use std::{
    io::{self, ErrorKind},
    ops::{Deref, DerefMut},
};

use crate::{types::Timestamp, utils};
use anchor_lang::prelude::*;
use solana_program::keccak::Hash;
use wormhole_solana_common::{legacy_account, NewAccountSize, SeedPrefix};
use wormhole_vaas::payloads::TypePrefixedPayload;

const SEED_PREFIX: &[u8] = b"PostedVAA";

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct PostedVaaV1Metadata {
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,

    /// Time the message was submitted.
    pub timestamp: Timestamp,

    /// Pubkey of `SignatureSet` account that represent this VAA's signature verification.
    pub signature_set: Pubkey,

    /// Guardian set index used to verify signatures for `SignatureSet`.
    ///
    /// NOTE: In the previous implementation, this member was referred to as the `posted_timestamp`,
    /// which is zero for VAA data (posted messages and VAAs resemble the same account schema). By
    /// changing this to the guardian set index, we patch a bug with verifying governance VAAs for
    /// the Core Bridge (other Core Bridge implementations require that the guardian set that
    /// attested for the governance VAA is the current one).
    pub guardian_set_index: u32,

    /// Unique id for this message.
    pub nonce: u32,

    /// Sequence number of this message.
    pub sequence: u64,

    /// Emitter of the message.
    pub emitter_chain: u16,

    /// Emitter of the message.
    pub emitter_address: [u8; 32],
}

pub trait VaaV1LegacyAccount: SeedPrefix {
    /// This discriminator used to be just "vaa" but now we encode the VAA version into the
    /// discriminator (instead of having a separate field in the account data's schema).
    const LEGACY_DISCRIMINATOR: [u8; 4] = *b"vaa\x01";

    /// Recompute the message hash, which can be used to derive the PostedVaa PDA address.
    ///
    /// NOTE: For a cheaper derivation, your instruction handler can take a message hash as an
    /// argument. But at the end of the day, re-hashing isn't that expensive.
    fn try_message_hash(&self) -> Result<Hash>;
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct MessagePayload<P: TypePrefixedPayload> {
    pub size: u32,
    pub data: Box<P>,
}

impl<P: TypePrefixedPayload> Deref for MessagePayload<P> {
    type Target = P;

    fn deref(&self) -> &Self::Target {
        &self.data
    }
}

impl<P: TypePrefixedPayload> DerefMut for MessagePayload<P> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.data
    }
}

impl<P: TypePrefixedPayload> AnchorDeserialize for MessagePayload<P> {
    fn deserialize(buf: &mut &[u8]) -> io::Result<Self> {
        let size = u32::deserialize(buf)?;

        // We check that the encoded size matches the actual size of the buffer.
        if usize::try_from(size).unwrap() != buf.len() {
            return Err(io::Error::new(
                ErrorKind::InvalidData,
                "invalid payload size",
            ));
        }

        Ok(Self {
            size,
            data: Box::new(P::read(buf)?),
        })
    }
}

impl<P: TypePrefixedPayload> AnchorSerialize for MessagePayload<P> {
    fn serialize<W: io::Write>(&self, _writer: &mut W) -> io::Result<()> {
        // NOTE: We only intend to read these payloads. Serialization only matters when we write
        // to an account that uses `MessagePayload<P>`.
        Ok(())
    }
}

#[legacy_account]
pub struct PostedVaaV1<P: TypePrefixedPayload> {
    pub meta: PostedVaaV1Metadata,
    pub payload: MessagePayload<P>,
}

impl<P: TypePrefixedPayload> SeedPrefix for PostedVaaV1<P> {
    fn seed_prefix() -> &'static [u8] {
        SEED_PREFIX
    }
}

impl<P: TypePrefixedPayload> VaaV1LegacyAccount for PostedVaaV1<P> {
    fn try_message_hash(&self) -> Result<Hash> {
        let mut payload = Vec::with_capacity(self.payload.size.try_into().unwrap());
        self.payload.data.write(&mut payload)?;

        Ok(utils::compute_message_hash(
            self.timestamp,
            self.nonce,
            self.emitter_chain,
            &self.emitter_address,
            self.sequence,
            self.consistency_level,
            &payload,
        ))
    }
}

impl<P: TypePrefixedPayload> Deref for PostedVaaV1<P> {
    type Target = PostedVaaV1Metadata;

    fn deref(&self) -> &Self::Target {
        &self.meta
    }
}

#[legacy_account]
pub(crate) struct PostedVaaV1Bytes {
    pub meta: PostedVaaV1Metadata,
    pub payload: Vec<u8>,
}

impl SeedPrefix for PostedVaaV1Bytes {
    fn seed_prefix() -> &'static [u8] {
        SEED_PREFIX
    }
}

impl VaaV1LegacyAccount for PostedVaaV1Bytes {
    fn try_message_hash(&self) -> Result<Hash> {
        Ok(utils::compute_message_hash(
            self.timestamp,
            self.nonce,
            self.emitter_chain,
            &self.emitter_address,
            self.sequence,
            self.consistency_level,
            &self.payload,
        ))
    }
}

impl Deref for PostedVaaV1Bytes {
    type Target = PostedVaaV1Metadata;

    fn deref(&self) -> &Self::Target {
        &self.meta
    }
}

impl NewAccountSize for PostedVaaV1Bytes {
    fn compute_size(payload_len: usize) -> usize {
        4 // LEGACY_DISCRIMINATOR
        + PostedVaaV1Metadata::INIT_SPACE
        + 4 // payload.len()
        + payload_len
    }
}
