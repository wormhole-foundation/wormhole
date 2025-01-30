use solana_program::{
    instruction::{AccountMeta, Instruction},
    pubkey::Pubkey,
};

use super::VerifyVaaShimInstruction;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CloseSignaturesAccounts<'ix> {
    pub guardian_signatures: &'ix Pubkey,

    pub refund_recipient: &'ix Pubkey,
}

/// Allows the initial payer to close the signature account, reclaiming the rent
/// taken by the post signatures instruction.
pub struct CloseSignatures<'ix> {
    pub program_id: &'ix Pubkey,
    pub accounts: CloseSignaturesAccounts<'ix>,
}

impl<'ix> CloseSignatures<'ix> {
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
