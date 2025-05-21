use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};
use wormhole_svm_definitions::GUARDIAN_SIGNATURE_LENGTH;

use super::VerifyVaaShimInstruction;

/// Accounts for the post signatures instruction.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostSignaturesAccounts<'ix> {
    pub payer: &'ix Pubkey,

    pub guardian_signatures: &'ix Pubkey,
}

/// Instruction data for the post signatures instruction.
///
/// Being contiguous is a feature that allows for the guardian signatures to be
/// stored in a single slice, which is more efficient for the SVM runtime.
/// When instruction data is initialized via [PostSignaturesData::new], the
/// data is not guaranteed to be contiguous.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PostSignaturesData<'ix, const CONTIGUOUS: bool> {
    /// Argument to encode the guardian set index associated with the guardian
    /// signatures.
    pub(super) guardian_set_index: u32,

    /// Total expected number of signatures, which determines the total size of
    /// the guardian signatures account.
    pub(super) total_signatures: u8,

    /// Guardian signatures to load into the guardian signatures account at this
    /// call.
    pub(super) guardian_signatures: &'ix [[u8; GUARDIAN_SIGNATURE_LENGTH]],
}

impl<'ix, const CONTIGUOUS: bool> PostSignaturesData<'ix, CONTIGUOUS> {
    pub const MINIMUM_SIZE: usize = {
        4 // guardian_set_index
        + 1 // total_signatures
        + 4 // guardian_signatures length
    };

    #[inline]
    pub fn guardian_set_index(&self) -> u32 {
        self.guardian_set_index
    }

    #[inline]
    pub fn total_signatures(&self) -> u8 {
        self.total_signatures
    }

    #[inline]
    pub fn guardian_signatures(&self) -> &'ix [[u8; GUARDIAN_SIGNATURE_LENGTH]] {
        self.guardian_signatures
    }
}

impl<'ix> PostSignaturesData<'ix, false> {
    pub fn new(
        guardian_set_index: u32,
        total_signatures: u8,
        guardian_signatures: &'ix [[u8; GUARDIAN_SIGNATURE_LENGTH]],
    ) -> Self {
        Self {
            guardian_set_index,
            total_signatures,
            guardian_signatures,
        }
    }
}

impl<'ix> PostSignaturesData<'ix, true> {
    #[inline(always)]
    pub(super) fn deserialize(data: &'ix [u8]) -> Option<Self> {
        if data.len() < Self::MINIMUM_SIZE {
            return None;
        }

        let guardian_set_index = u32::from_le_bytes(data[..4].try_into().unwrap());
        let total_signatures = data[4];
        let guardian_signatures_len = u32::from_le_bytes(data[5..9].try_into().unwrap()) as usize;

        let encoded_signatures_len =
            guardian_signatures_len.checked_mul(GUARDIAN_SIGNATURE_LENGTH)?;
        let total_len = Self::MINIMUM_SIZE.checked_add(encoded_signatures_len)?;

        if data.len() < total_len {
            return None;
        }

        let guardian_signatures = &data[9..total_len];

        // Safety: Guardian signatures are contiguous and its length is a
        // multiple of SIGNATURE_LENGTH.
        let guardian_signatures = unsafe {
            core::slice::from_raw_parts(
                guardian_signatures.as_ptr() as *const [u8; GUARDIAN_SIGNATURE_LENGTH],
                guardian_signatures_len,
            )
        };

        // NOTE: We do not care about trailing bytes.

        Some(Self {
            guardian_set_index,
            total_signatures,
            guardian_signatures,
        })
    }

    #[inline]
    pub fn guardian_signatures_slice(&self) -> &'ix [u8] {
        // Safety: Guardian signatures are contiguous and its length is a
        // multiple of SIGNATURE_LENGTH.
        unsafe {
            core::slice::from_raw_parts(
                self.guardian_signatures.as_ptr() as *const u8,
                self.guardian_signatures.len() * GUARDIAN_SIGNATURE_LENGTH,
            )
        }
    }
}

/// Creates or appends to a guardian signatures account for subsequent use by
/// the verify hash instruction.
///
/// This instruction is necessary due to the Wormhole VAA body, which has an
/// arbitrary size, and 13 guardian signatures (a quorum of the current 19
/// mainnet guardians, 66 bytes each) alongside the required accounts is likely
/// larger than the transaction size limit on Solana (1232 bytes).
///
/// This instruction will also allow for the verification of other messages
/// which guardians sign, such as query results.
///
/// This instruction allows for the initial payer to append additional
/// signatures to the account by calling the instruction again. Subsequent
/// calls may be necessary if a quorum of signatures from the current guardian
/// set grows larger than can fit into a single transaction.
///
/// The guardian signatures account can be closed by the initial payer via the
/// close signatures instruction, which will refund this payer.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PostSignatures<'ix> {
    pub program_id: &'ix Pubkey,
    pub accounts: PostSignaturesAccounts<'ix>,
    pub data: PostSignaturesData<'ix, false>,
}

impl PostSignatures<'_> {
    /// Generate SVM instruction.
    #[inline]
    pub fn instruction(&self) -> Instruction {
        Instruction {
            program_id: *self.program_id,
            accounts: vec![
                AccountMeta::new(*self.accounts.payer, true),
                AccountMeta::new(*self.accounts.guardian_signatures, true),
                AccountMeta::new_readonly(solana_program::system_program::ID, false),
            ],
            data: VerifyVaaShimInstruction::PostSignatures(self.data).to_vec(),
        }
    }
}
