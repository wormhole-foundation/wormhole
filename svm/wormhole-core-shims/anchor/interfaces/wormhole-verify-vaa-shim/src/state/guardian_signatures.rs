use anchor_lang::prelude::*;

#[account]
#[derive(Debug)]
pub struct GuardianSignatures {
    /// Payer of this guardian signatures account.
    /// Only they may amend signatures.
    /// Used for reimbursements upon cleanup.
    pub refund_recipient: Pubkey,

    /// Guardian set index that these signatures correspond to.
    /// Storing this simplifies the integrator data.
    /// Using big-endian to match the derivation used by the core bridge.
    pub guardian_set_index_be: [u8; 4],

    /// Unverified guardian signatures.
    pub guardian_signatures: Vec<[u8; 66]>,
}
