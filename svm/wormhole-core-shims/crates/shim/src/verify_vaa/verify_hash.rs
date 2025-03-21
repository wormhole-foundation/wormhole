use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};

use super::{Hash, VerifyVaaShimInstruction};

/// Accounts for the verify hash instruction.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct VerifyHashAccounts<'ix> {
    /// Guardian set is used to verify the recovered public keys from the
    /// signatures found in the guardian signatures account and
    /// [VerifyHashData::digest].
    pub guardian_set: &'ix Pubkey,

    /// Guardian signatures account created using [PostSignatures].
    ///
    /// [PostSignatures]: super::PostSignatures
    pub guardian_signatures: &'ix Pubkey,
}

/// Instruction data for the verify hash instruction.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct VerifyHashData {
    pub(super) guardian_set_bump: u8,
    pub(super) digest: Hash,
}

impl VerifyHashData {
    pub const SIZE: usize = {
        1 // guardian set bump
        + 32 // digest
    };

    pub fn new(guardian_set_bump: u8, digest: Hash) -> Self {
        Self {
            guardian_set_bump,
            digest,
        }
    }

    #[inline]
    pub fn guardian_set_bump(&self) -> u8 {
        self.guardian_set_bump
    }

    #[inline]
    pub fn digest(&self) -> Hash {
        self.digest
    }

    #[inline(always)]
    pub(super) fn deserialize(data: &[u8]) -> Option<Self> {
        if data.len() < Self::SIZE {
            return None;
        }

        Some(Self {
            guardian_set_bump: data[0],
            digest: Hash(data[1..33].try_into().unwrap()),
        })
    }
}

/// This instruction is intended to be invoked via CPI call. It verifies a
/// digest against a guardian signatures account and a Wormhole Core Bridge
/// guardian set account.
///
/// Prior to this call (and likely in a separate transaction), call the post
/// signatures instruction to create the guardian signatures account.
///
/// Immediately after this verify call, call the close signatures instruction
/// to reclaim the rent paid to create the guardian signatures account.
///
/// A v1 VAA digest can be computed as follows:
/// ```rust
/// use wormhole_svm_definitions::compute_keccak_digest;
///
/// // `vec_body` is the encoded body of the VAA.
/// # let vaa_body = vec![];
/// let digest = compute_keccak_digest(
///     solana_program::keccak::hash(&vaa_body),
///     None, // there is no prefix for V1 messages
/// );
/// ```
///
/// A QueryResponse digest can be computed as follows:
/// ```rust
/// # mod wormhole_query_sdk {
/// #    pub const MESSAGE_PREFIX: &'static [u8] = b"ruh roh";
/// # }
/// use wormhole_query_sdk::MESSAGE_PREFIX;
/// use wormhole_svm_definitions::compute_keccak_digest;
///
/// # let query_response_bytes = vec![];
/// let digest = compute_keccak_digest(
///     solana_program::keccak::hash(&query_response_bytes),
///     Some(MESSAGE_PREFIX)
/// );
/// ```
pub struct VerifyHash<'ix> {
    pub program_id: &'ix Pubkey,
    pub accounts: VerifyHashAccounts<'ix>,
    pub data: VerifyHashData,
}

impl VerifyHash<'_> {
    /// Generate SVM instruction.
    #[inline]
    pub fn instruction(&self) -> Instruction {
        Instruction {
            program_id: *self.program_id,
            accounts: vec![
                AccountMeta::new_readonly(*self.accounts.guardian_set, false),
                AccountMeta::new_readonly(*self.accounts.guardian_signatures, false),
            ],
            data: VerifyVaaShimInstruction::VerifyHash(self.data).to_vec(),
        }
    }
}
