use anchor_lang::prelude::*;
use wormhole_svm_definitions::HASH_BYTES;

declare_id!("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

#[program]
pub mod wormhole_verify_vaa_shim {
    use super::*;

    /// Allows the initial payer to close the signature account, reclaiming the
    /// rent taken by the post signatures instruction.
    pub fn close_signatures(_ctx: Context<CloseSignatures>) -> Result<()> {
        err!(ErrorCode::InstructionMissing)
    }

    /// Creates or appends to a guardian signatures account for subsequent use
    /// by the verify hash instruction.
    ///
    /// This instruction is necessary due to the Wormhole VAA body, which has an
    /// arbitrary size, and 13 guardian signatures (a quorum of the current 19
    /// mainnet guardians, 66 bytes each) alongside the required accounts is
    /// likely larger than the transaction size limit on Solana (1232 bytes).
    ///
    /// This instruction will also allow for the verification of other messages
    /// which guardians sign, such as query results.
    ///
    /// This instruction allows for the initial payer to append additional
    /// signatures to the account by calling the instruction again. Subsequent
    /// calls may be necessary if a quorum of signatures from the current guardian
    /// set grows larger than can fit into a single transaction.
    ///
    /// The guardian signatures account can be closed by the initial payer via
    /// the close signatures instruction, which will refund this payer.
    pub fn post_signatures(
        _ctx: Context<PostSignatures>,
        guardian_set_index: u32,
        total_signatures: u8,
        guardian_signatures: Vec<[u8; 66]>,
    ) -> Result<()> {
        let _ = (guardian_set_index, total_signatures, guardian_signatures);
        err!(ErrorCode::InstructionMissing)
    }

    /// This instruction is intended to be invoked via CPI call. It verifies a
    /// digest against a guardian signatures account and a Wormhole Core Bridge
    /// guardian set account.
    ///
    /// Prior to this call (and likely in a separate transaction), call the post
    /// signatures instruction to create the guardian signatures account.
    ///
    /// Immediately after this verify call, call the close signatures
    /// instruction to reclaim the rent paid to create the guardian signatures
    /// account.
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
    pub fn verify_hash(
        _ctx: Context<VerifyHash>,
        guardian_set_bump: u8,
        digest: [u8; HASH_BYTES],
    ) -> Result<()> {
        let _ = (guardian_set_bump, digest);
        err!(ErrorCode::InstructionMissing)
    }
}

#[derive(Accounts)]
pub struct CloseSignatures<'info> {
    #[account(mut, has_one = refund_recipient)]
    guardian_signatures: Account<'info, GuardianSignatures>,

    #[account(mut, address = guardian_signatures.refund_recipient)]
    refund_recipient: Signer<'info>,
}

#[derive(Accounts)]
pub struct PostSignatures<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(mut, signer)]
    guardian_signatures: Account<'info, GuardianSignatures>,

    system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct VerifyHash<'info> {
    /// Guardian set used for signature verification.
    guardian_set: UncheckedAccount<'info>,

    /// Stores unverified guardian signatures as they are too large to fit in
    /// the instruction data.
    guardian_signatures: Account<'info, GuardianSignatures>,
}

#[account]
#[derive(Debug, PartialEq, Eq)]
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
