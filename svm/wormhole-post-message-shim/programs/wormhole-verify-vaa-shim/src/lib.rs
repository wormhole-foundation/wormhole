use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::HASH_BYTES;

declare_id!("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

mod instructions;
pub(crate) use instructions::*;

pub mod state;

pub mod error;

#[program]
pub mod wormhole_verify_vaa_shim {

    use super::*;

    /// Allows the initial payer to close the signature account, reclaiming the rent taken by `post_signatures`.
    pub fn close_signatures(ctx: Context<CloseSignatures>) -> Result<()> {
        instructions::close_signatures(ctx)
    }

    /// Creates or appends to a GuardianSignatures account for subsequent use by verify_vaa.
    /// This is necessary as the Wormhole VAA body, which has an arbitrary size,
    /// and 13 guardian signatures (a quorum of the current 19 mainnet guardians, 66 bytes each)
    /// alongside the required accounts is likely larger than the transaction size limit on Solana (1232 bytes).
    /// This will also allow for the verification of other messages which guardians sign, such as QueryResults.
    ///
    /// This instruction allows for the initial payer to append additional signatures to the account by calling the instruction again.
    /// This may be necessary if a quorum of signatures from the current guardian set grows larger than can fit into a single transaction.
    ///
    /// The GuardianSignatures account can be closed by the initial payer via close_signatures, which will refund the initial payer.
    pub fn post_signatures(
        ctx: Context<PostSignatures>,
        guardian_set_index: u32,
        total_signatures: u8,
        guardian_signatures: Vec<[u8; 66]>,
    ) -> Result<()> {
        instructions::post_signatures(
            ctx,
            guardian_set_index,
            total_signatures,
            guardian_signatures,
        )
    }

    /// This instruction is intended to be invoked via CPI call. It verifies a digest against a GuardianSignatures account
    /// and a core bridge GuardianSet.
    /// Prior to this call, and likely in a separate transaction, `post_signatures` must be called to create the account.
    /// Immediately after this call, `close_signatures` should be called to reclaim the lamports.
    ///
    /// A v1 VAA digest can be computed as follows:
    /// ```rust
    ///   let message_hash = &solana_program::keccak::hashv(&[&vaa_body]).to_bytes();
    ///   let digest = keccak::hash(message_hash.as_slice()).to_bytes();
    /// ```
    ///
    /// A QueryResponse digest can be computed as follows:
    /// ```rust
    ///   use wormhole_query_sdk::MESSAGE_PREFIX;
    ///   let message_hash = [
    ///     MESSAGE_PREFIX,
    ///     &solana_program::keccak::hashv(&[&bytes]).to_bytes(),
    ///   ]
    ///   .concat();
    ///   let digest = keccak::hash(message_hash.as_slice()).to_bytes();
    /// ```
    pub fn verify_vaa(ctx: Context<VerifyVaa>, digest: [u8; HASH_BYTES]) -> Result<()> {
        instructions::verify_vaa(ctx, digest)
    }
}
