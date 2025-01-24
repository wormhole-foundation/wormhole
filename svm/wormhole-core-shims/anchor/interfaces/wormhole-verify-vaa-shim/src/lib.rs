use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::HASH_BYTES;
use wormhole_solana_consts::CORE_BRIDGE_PROGRAM_ID;

declare_id!("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

pub mod state;
use state::*;

#[program]
pub mod wormhole_verify_vaa_shim {

    use super::*;

    /// Allows the initial payer to close the signature account, reclaiming the rent taken by `post_signatures`.
    pub fn close_signatures(_ctx: Context<CloseSignatures>) -> Result<()> {
        err!(ErrorCode::InstructionMissing)
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
        _ctx: Context<PostSignatures>,
        _guardian_set_index: u32,
        _total_signatures: u8,
        _guardian_signatures: Vec<[u8; 66]>,
    ) -> Result<()> {
        err!(ErrorCode::InstructionMissing)
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
    pub fn verify_vaa(_ctx: Context<VerifyVaa>, _digest: [u8; HASH_BYTES]) -> Result<()> {
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
pub struct VerifyVaa<'info> {
    /// Guardian set used for signature verification.
    #[account(
        seeds = [
            WormholeGuardianSet::SEED_PREFIX,
            guardian_signatures.guardian_set_index_be.as_ref()
        ],
        bump,
        seeds::program = CORE_BRIDGE_PROGRAM_ID
    )]
    guardian_set: Account<'info, WormholeGuardianSet>,

    /// Stores unverified guardian signatures as they are too large to fit in the instruction data.
    guardian_signatures: Account<'info, GuardianSignatures>,
}
