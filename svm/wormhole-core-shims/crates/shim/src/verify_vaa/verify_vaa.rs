use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};

use super::{Hash, VerifyVaaShimInstruction};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct VerifyVaaAccounts<'ix> {
    pub guardian_set: &'ix Pubkey,

    pub guardian_signatures: &'ix Pubkey,
}

/// This instruction is intended to be invoked via CPI call. It verifies a
/// digest against a guardian signatures account and a Wormhole Core Bridge
/// guardian set account.
///
/// Prior to this call (and likely in a separate transaction), call the post
/// signatures instruction to create the guardian signatures account.
///
/// Immediately after this verify call, call the close signatures instruction
/// to reclaim the rent paid to create the guardian signatures account
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
pub struct VerifyVaa<'ix> {
    pub program_id: &'ix Pubkey,
    pub accounts: VerifyVaaAccounts<'ix>,
    pub data: Hash,
}

impl<'ix> VerifyVaa<'ix> {
    pub fn instruction(&self) -> Instruction {
        Instruction {
            program_id: *self.program_id,
            accounts: vec![
                AccountMeta::new_readonly(*self.accounts.guardian_set, false),
                AccountMeta::new_readonly(*self.accounts.guardian_signatures, false),
            ],
            data: VerifyVaaShimInstruction::VerifyVaa(self.data).to_vec(),
        }
    }
}
