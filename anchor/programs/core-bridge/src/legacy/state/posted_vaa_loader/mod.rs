//! Type facilitating on demand zero copy deserialization.

use anchor_lang::prelude::*;
use solana_program::{account_info::AccountInfo, instruction::AccountMeta, keccak, pubkey::Pubkey};
use wormhole_solana_common::{LegacyDiscriminator, SeedPrefix};

use std::collections::{BTreeMap, BTreeSet};
use std::fmt;

use crate::types::Timestamp;

#[derive(Clone)]
pub struct PostedVaaV1Loader<'info> {
    acc_info: AccountInfo<'info>,
}

impl LegacyDiscriminator<4> for PostedVaaV1Loader<'_> {
    const LEGACY_DISCRIMINATOR: [u8; 4] = *b"vaa\x01";
}

impl SeedPrefix for PostedVaaV1Loader<'_> {
    fn seed_prefix() -> &'static [u8] {
        b"PostedVAA"
    }
}

impl<'info> fmt::Debug for PostedVaaV1Loader<'info> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("PostedVaaV1Loader")
            .field("acc_info", &self.acc_info)
            .finish()
    }
}

impl<'info> PostedVaaV1Loader<'info> {
    pub fn new(acc_info: AccountInfo<'info>) -> PostedVaaV1Loader<'info> {
        Self { acc_info }
    }

    /// Constructs a new `Loader` from a previously initialized account.
    #[inline(never)]
    pub fn try_from(acc_info: &AccountInfo<'info>) -> Result<PostedVaaV1Loader<'info>> {
        if *acc_info.owner != crate::ID {
            return Err(Error::from(ErrorCode::AccountOwnedByWrongProgram)
                .with_pubkeys((*acc_info.owner, crate::ID)));
        }
        let mut data: &[u8] = &acc_info.try_borrow_data()?;
        if data.len() < Self::LEGACY_DISCRIMINATOR.len() {
            return Err(ErrorCode::AccountDiscriminatorNotFound.into());
        }
        // Discriminator must match.
        let disc_bytes = <[u8; 4]>::deserialize(&mut data)?;
        if disc_bytes != Self::LEGACY_DISCRIMINATOR {
            return Err(ErrorCode::AccountDiscriminatorMismatch.into());
        }

        Ok(PostedVaaV1Loader::new(acc_info.clone()))
    }

    pub fn try_consistency_level(&self) -> Result<u8> {
        try_consistency_level(&self.acc_info)
    }

    pub fn try_timestamp(&self) -> Result<Timestamp> {
        try_consistency_level(&self.acc_info)
    }

    pub fn try_signature_set(&self) -> Result<Pubkey> {
        try_consistency_level(&self.acc_info)
    }

    pub fn try_guardian_set_index(&self) -> Result<u32> {
        try_guardian_set_index(&self.acc_info)
    }

    pub fn try_nonce(&self) -> Result<u32> {
        try_nonce(&self.acc_info)
    }

    pub fn try_sequence(&self) -> Result<u64> {
        try_sequence(&self.acc_info)
    }

    pub fn try_emitter_chain(&self) -> Result<u16> {
        try_emitter_chain(&self.acc_info)
    }

    pub fn try_emitter_address(&self) -> Result<[u8; 32]> {
        try_emitter_address(&self.acc_info)
    }

    pub fn try_message_hash(&self) -> Result<keccak::Hash> {
        let data = self.acc_info.try_borrow_data()?;
        Ok(keccak::hashv(&[
            data[5..9].as_ref(),   // timestamp
            data[45..49].as_ref(), // nonce
            data[57..59].as_ref(), // emitter_chain
            data[59..91].as_ref(), // emitter_address
            data[49..57].as_ref(), // sequence
            &[data[4]],            // consistency_level
            &data[95..],           // payload
        ]))
    }
}

pub fn try_consistency_level(&acc_info: &AccountInfo) -> Result<u8> {
    let data: &[u8] = &acc_info.try_borrow_data()?;
    Ok(data[4])
}

pub fn try_timestamp(&acc_info: &AccountInfo) -> Result<Timestamp> {
    let data = &acc_info.try_borrow_data()?;
    Ok(u32::from_le_bytes(data[5..9].try_into().unwrap()).into())
}

pub fn try_signature_set(&acc_info: &AccountInfo) -> Result<Pubkey> {
    let data = &acc_info.try_borrow_data()?;
    Ok(data[9..41].try_into().unwrap())
}

pub fn try_guardian_set_index(&acc_info: &AccountInfo) -> Result<u32> {
    let data = &acc_info.try_borrow_data()?;
    Ok(u32::from_le_bytes(data[41..45].try_into().unwrap()))
}

pub fn try_nonce(&acc_info: &AccountInfo) -> Result<u32> {
    let data = &acc_info.try_borrow_data()?;
    Ok(u32::from_le_bytes(data[45..49].try_into().unwrap()))
}

pub fn try_sequence(&acc_info: &AccountInfo) -> Result<u64> {
    let data = &acc_info.try_borrow_data()?;
    Ok(u64::from_le_bytes(data[49..57].try_into().unwrap()))
}

pub fn try_emitter_chain(&acc_info: &AccountInfo) -> Result<u16> {
    let data = &acc_info.try_borrow_data()?;
    Ok(u16::from_le_bytes(data[57..59].try_into().unwrap()))
}

pub fn try_emitter_address(&acc_info: &AccountInfo) -> Result<[u8; 32]> {
    let data = &acc_info.try_borrow_data()?;
    Ok(data[59..91].try_into().unwrap())
}
