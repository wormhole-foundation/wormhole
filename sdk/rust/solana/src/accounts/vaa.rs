//! VAA type specific to Solana.
//!
//! Solana's VAA kind represents a VAA "after" it has been processed by the bridge, it differs in a
//! two minor ways specific to Solana:
//!
//! - The field order differs from the Wormhole VAA wire format.
//! - Rather than include Signatures directly, it has a Pubkey to a signature account.
//!
//! It forms the basis of both Message and VAA accounts on Solana, but is still trivially
//! convertible back to the core VAA type in the SDK.

use {
    super::Account,
    borsh::{
        BorshDeserialize,
        BorshSerialize,
    },
    solana_program::{
        account_info::AccountInfo,
        pubkey::Pubkey,
    },
    wormhole::WormholeError,
};

#[derive(Debug, Eq, PartialEq, BorshSerialize, BorshDeserialize, Clone)]
pub struct VAA {
    /// Header of the posted VAA
    pub vaa_version:           u8,
    /// Level of consistency requested by the emitter
    pub consistency_level:     u8,
    /// Time the vaa was submitted
    pub vaa_time:              u32,
    /// Account where signatures are stored
    pub vaa_signature_account: Pubkey,
    /// Time the posted message was created
    pub submission_time:       u32,
    /// Unique nonce for this message
    pub nonce:                 u32,
    /// Sequence number of this message
    pub sequence:              u64,
    /// Emitter of the message
    pub emitter_chain:         u16,
    /// Emitter of the message
    pub emitter_address:       [u8; 32],
    /// Message payload
    pub payload:               Vec<u8>,
}

impl Account for VAA {
    type Seeds = [u8; 32];
    type Output = Pubkey;

    fn key(id: &Pubkey, vaa_hash: [u8; 32]) -> Pubkey {
        let (vaa, _) = Pubkey::find_program_address(&[b"PostedVAA", &vaa_hash], id);
        vaa
    }

    fn get(account: &AccountInfo) -> Result<Self, WormholeError> {
        VAA::try_from_slice(&account.data.borrow()).map_err(|_| WormholeError::DeserializeFailed)
    }
}
