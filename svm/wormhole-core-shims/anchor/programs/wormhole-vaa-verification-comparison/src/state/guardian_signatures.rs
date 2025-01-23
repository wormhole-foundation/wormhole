use anchor_lang::prelude::*;

#[account]
#[derive(Debug)]
pub struct GuardianSignatures {
    /// Payer of this guardian signatures account.
    /// Only they may amend signatures.
    /// Used for reimbursements upon cleanup.
    pub refund_recipient: Pubkey,

    /// Unverified guardian signatures.
    pub guardian_signatures: Vec<[u8; 66]>,
}

impl GuardianSignatures {
    pub(crate) fn compute_size(num_guardians: usize) -> usize {
        32 // refund_recipient
        + 4 + num_guardians * 66 // signatures
    }

    pub fn is_initialized(&self) -> bool {
        !self.guardian_signatures.is_empty()
    }
}
