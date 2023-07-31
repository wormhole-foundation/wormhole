use std::{
    io::{self, ErrorKind},
    ops::{Deref, DerefMut},
};

use crate::{types::Timestamp, utils};
use anchor_lang::prelude::*;
use solana_program::keccak;
use wormhole_io::{Readable, Writeable};
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, NewAccountSize, SeedPrefix};

const SEED_PREFIX: &[u8] = b"PostedVAA";
const LEGACY_DISCRIMINATOR: [u8; 4] = *b"vaa\x01";

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
    const LEGACY_DISCRIMINATOR: [u8; 4] = LEGACY_DISCRIMINATOR;

    /// Recompute the message hash, which can be used to derive the PostedVaa PDA address.
    ///
    /// NOTE: For a cheaper derivation, your instruction handler can take a message hash as an
    /// argument. But at the end of the day, re-hashing isn't that expensive.
    fn try_message_hash(&self) -> Result<keccak::Hash>;
}

// #[derive(Debug, Clone, Eq, PartialEq)]
// pub struct MessagePayload<P: Readable + Writeable> {
//     pub size: u32,
//     pub data: Box<P>,
// }

// impl<P: Readable + Writeable> Deref for MessagePayload<P> {
//     type Target = P;

//     fn deref(&self) -> &Self::Target {
//         &self.data
//     }
// }

// impl<P: Readable + Writeable> DerefMut for MessagePayload<P> {
//     fn deref_mut(&mut self) -> &mut Self::Target {
//         &mut self.data
//     }
// }

// impl<P: Readable + Writeable> AnchorDeserialize for MessagePayload<P> {
//     fn deserialize(buf: &mut &[u8]) -> io::Result<Self> {
//         let size = u32::deserialize(buf)?;

//         // We check that the encoded size matches the actual size of the buffer.
//         if usize::try_from(size).unwrap() != buf.len() {
//             return Err(io::Error::new(
//                 ErrorKind::InvalidData,
//                 "invalid payload size",
//             ));
//         }

//         Ok(Self {
//             size,
//             data: Box::new(P::read(buf)?),
//         })
//     }
// }

// impl<P: Readable + Writeable> AnchorSerialize for MessagePayload<P> {
//     fn serialize<W: io::Write>(&self, _writer: &mut W) -> io::Result<()> {
//         // NOTE: We only intend to read these payloads. Serialization only matters when we write
//         // to an account that uses `MessagePayload<P>`.
//         Ok(())
//     }
// }

// #[legacy_account]
// pub struct PostedVaaV1<P: Readable + Writeable> {
//     pub meta: PostedVaaV1Metadata,
//     pub payload: MessagePayload<P>,
// }

// impl<P: Readable + Writeable> SeedPrefix for PostedVaaV1<P> {
//     fn seed_prefix() -> &'static [u8] {
//         SEED_PREFIX
//     }
// }

// impl<P: Readable + Writeable> VaaV1LegacyAccount for PostedVaaV1<P> {
//     fn try_message_hash(&self) -> Result<keccak::Hash> {
//         let mut payload = Vec::with_capacity(self.payload.size.try_into().unwrap());
//         self.payload.data.write(&mut payload)?;

//         // Ok(utils::compute_message_hash(
//         //     self.timestamp,
//         //     self.nonce,
//         //     self.emitter_chain,
//         //     &self.emitter_address,
//         //     self.sequence,
//         //     self.consistency_level,
//         //     &payload,
//         // ))
//         Ok(keccak::hashv(&[
//             &self.timestamp.to_be_bytes(),     // timestamp
//             &self.nonce.to_be_bytes(),         // nonce
//             &self.emitter_chain.to_be_bytes(), // emitter_chain
//             &self.emitter_address,             // emitter_address
//             &self.sequence.to_be_bytes(),      // sequence
//             &[self.consistency_level],         // consistency_level
//             &payload,                          // payload
//         ]))
//     }
// }

// impl<P: Readable + Writeable> Deref for PostedVaaV1<P> {
//     type Target = PostedVaaV1Metadata;

//     fn deref(&self) -> &Self::Target {
//         &self.meta
//     }
// }

#[legacy_account]
pub struct PostedVaaV1Bytes {
    pub meta: PostedVaaV1Metadata,
    pub payload: Vec<u8>,
}

impl SeedPrefix for PostedVaaV1Bytes {
    fn seed_prefix() -> &'static [u8] {
        SEED_PREFIX
    }
}

impl VaaV1LegacyAccount for PostedVaaV1Bytes {
    fn try_message_hash(&self) -> Result<keccak::Hash> {
        Ok(keccak::hashv(&[
            &self.timestamp.to_be_bytes(),     // timestamp
            &self.nonce.to_be_bytes(),         // nonce
            &self.emitter_chain.to_be_bytes(), // emitter_chain
            &self.emitter_address,             // emitter_address
            &self.sequence.to_be_bytes(),      // sequence
            &[self.consistency_level],         // consistency_level
            &self.payload,                     // payload
        ]))
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

#[legacy_account]
#[derive(InitSpace)]
pub(crate) struct PartialPostedVaaV1 {
    pub meta: PostedVaaV1Metadata,
    pub payload_len: u32,
}

impl PartialPostedVaaV1 {
    pub const PAYLOAD_START: usize = 95;

    pub fn try_message_hash(acct: &Account<Self>) -> Result<keccak::Hash> {
        let acc_info: &AccountInfo = acct.as_ref();
        let data: &[u8] = &acc_info.try_borrow_data()?;

        let timestamp = u32::from_le_bytes(data[5..9].try_into().unwrap()).to_be_bytes();
        let nonce = u32::from_le_bytes(data[45..49].try_into().unwrap()).to_be_bytes();
        let emitter_chain = u16::from_le_bytes(data[57..59].try_into().unwrap()).to_be_bytes();
        let emitter_address = &data[59..91];
        let sequence = u64::from_le_bytes(data[49..57].try_into().unwrap()).to_be_bytes();
        let consistency_level = [data[4]];

        Ok(keccak::hashv(&[
            &timestamp,                                 // timestamp
            &nonce,                                     // nonce
            &emitter_chain,                             // emitter_chain
            emitter_address,                            // emitter_address
            &sequence,                                  // sequence
            &consistency_level,                         // consistency_level
            &data[PartialPostedVaaV1::PAYLOAD_START..], // payload
        ]))
    }
}

impl LegacyDiscriminator<4> for PartialPostedVaaV1 {
    const LEGACY_DISCRIMINATOR: [u8; 4] = *b"vaa\x01";
}

impl SeedPrefix for PartialPostedVaaV1 {
    fn seed_prefix() -> &'static [u8] {
        SEED_PREFIX
    }
}

impl Deref for PartialPostedVaaV1 {
    type Target = PostedVaaV1Metadata;

    fn deref(&self) -> &Self::Target {
        &self.meta
    }
}
