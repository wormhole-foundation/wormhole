use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};

use super::VerifyVaaShimInstruction;

/// Accounts for the close signatures instruction.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CloseSignaturesAccounts<'ix> {
    /// Guardian signatures account created using [PostSignatures]. This account
    /// will be closed and the rent refunded to the original payer.
    ///
    /// [PostSignatures]: super::PostSignatures
    pub guardian_signatures: &'ix Pubkey,

    /// Original payer of the guardian signatures account. This account will
    /// receive the rent refund when the guardian signatures account is closed.
    pub refund_recipient: &'ix Pubkey,
}

/// Allows the initial payer to close the signature account, reclaiming the rent
/// taken by the post signatures instruction.
pub struct CloseSignatures<'ix> {
    pub program_id: &'ix Pubkey,
    pub accounts: CloseSignaturesAccounts<'ix>,
}

impl CloseSignatures<'_> {
    /// Generate SVM instruction.
    pub fn instruction(&self) -> Instruction {
        Instruction {
            program_id: *self.program_id,
            accounts: vec![
                AccountMeta::new(*self.accounts.guardian_signatures, false),
                AccountMeta::new(*self.accounts.refund_recipient, true),
            ],
            data: VerifyVaaShimInstruction::CloseSignatures.to_vec(),
        }
    }
}
