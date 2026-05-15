use crate::{
    accounts::{
        Bridge,
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        PostedVAA,
        PostedVAADerivationData,
        SignatureSetData,
    },
    api::post_vaa::check_active,
    error::Error::{
        GuardianSetStillActive,
        InvalidDerivedAccount,
        InvalidProgramOwner,
        MathOverflow,
        MessageWithinRetentionWindow,
    },
    MessageData,
};
use solana_program::{
    program_error::ProgramError,
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::Seeded,
    *,
};

use super::RETENTION_PERIOD;

/// Default submission_time for accounts created before the submission_time
/// field was populated. Gives legacy accounts a 30-day grace period from
/// this date (10 April 2026) instead of being immediately closable.
const DEFAULT_SUBMISSION_TIME: u32 = 1775833078;

#[derive(FromAccounts)]
pub struct CloseSignatureSetAndPostedVAA<'b> {
    /// Bridge config, used to update last_lamports.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// The signature set account to close.
    ///
    /// Constraints (verified in the handler — SignatureSet has no magic
    /// prefix, so each one matters):
    ///   1. Owned by the core bridge program.
    ///   2. Bytes parse exactly into a `SignatureSetData` (try_from_slice
    ///      enforces both "no leftover bytes" and "no missing bytes"; this
    ///      is the primary defense against arbitrary-account-close attacks
    ///      since there is no discriminator).
    ///   3. Its `hash` derives the `posted_vaa` account passed below.
    ///   4. If `posted_vaa` is uninitialized: the `guardian_set` whose
    ///      index matches `sig_data.guardian_set_index` must be expired.
    ///
    /// Uses Info to avoid Persist serializing back into a zeroed account.
    pub signature_set: Mut<Info<'b>>,

    /// The PostedVAA account derived from the signature set's hash.
    /// May or may not be initialized. Uses Info for manual handling.
    pub posted_vaa: Mut<Info<'b>>,

    /// The guardian set used to verify the signature set.
    pub guardian_set: GuardianSet<'b, { AccountState::Initialized }>,

    /// Fee collector PDA to receive the reclaimed lamports.
    pub fee_collector: Mut<FeeCollector<'b>>,

    /// Clock for timestamp validation.
    pub clock: Sysvar<'b, Clock>,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct CloseSignatureSetAndPostedVAAData {}

pub fn close_signature_set_and_posted_vaa(
    ctx: &ExecutionContext,
    accs: &mut CloseSignatureSetAndPostedVAA,
    _data: CloseSignatureSetAndPostedVAAData,
) -> Result<()> {
    // 1. Verify signature_set is owned by this program.
    if accs.signature_set.owner != ctx.program_id {
        return Err(InvalidProgramOwner.into());
    }

    // 2. Parse SignatureSetData.
    let sig_data = {
        let data = accs.signature_set.data.borrow();
        SignatureSetData::try_from_slice(&data)
            .map_err(|_| SolitaireError::ProgramError(ProgramError::InvalidAccountData))?
    };

    // 3. Verify PostedVAA PDA derivation matches the hash from the SignatureSet.
    let msg_derivation = PostedVAADerivationData {
        payload_hash: sig_data.hash.to_vec(),
    };
    let expected_vaa_seeds =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::seeds(&msg_derivation);
    let expected_vaa_seeds_ref: Vec<&[u8]> =
        expected_vaa_seeds.iter().map(|s| s.as_slice()).collect();
    let (expected_vaa_key, _) =
        Pubkey::find_program_address(&expected_vaa_seeds_ref, ctx.program_id);
    if accs.posted_vaa.key != &expected_vaa_key {
        return Err(InvalidDerivedAccount.into());
    }

    // 4. Determine if PostedVAA is initialised. We use the same conjunction
    // (lamports + data + owner) the bridge uses elsewhere for "live" accounts;
    // after a close (below) all three become zero/system-owned at once.
    let vaa_initialized = accs.posted_vaa.lamports() > 0
        && accs.posted_vaa.data_len() > 0
        && accs.posted_vaa.owner == ctx.program_id;

    // 5. Branched validation.
    if vaa_initialized {
        // 5a. PostedVAA is initialised: check retention window.
        let vaa_data = {
            let data = accs.posted_vaa.data.borrow();
            if data.len() < 3 {
                return Err(SolitaireError::ProgramError(
                    ProgramError::InvalidAccountData,
                ));
            }
            // sanity check: the prefix should be "vaa" for a PostedVAA account
            // (technically redundant because we checked the derivation, but we
            // follow the principle of not silently dropping bytes even when we
            // think we know what they are).
            if &data[0..3] != b"vaa" {
                return Err(SolitaireError::ProgramError(
                    ProgramError::InvalidAccountData,
                ));
            }
            MessageData::try_from_slice(&data[3..])
                .map_err(|_| SolitaireError::ProgramError(ProgramError::InvalidAccountData))?
        };

        // Defense-in-depth shape checks for the PostedVAA. post_vaa fills all
        // three of these out of the verified VAA, so a posted_vaa that fails
        // any of them is either corrupt or a different account masquerading
        // under a "vaa" prefix — refuse to close it.
        //
        //   - vaa_version: legitimate VAAs are version 1 or higher.
        //   - vaa_time: legitimate VAAs always carry a non-zero timestamp.
        //   - vaa_signature_account: pins the close to the *specific*
        //     signature_set that verified this VAA, blocking pairings with
        //     unrelated signature sets that happen to share a payload hash.
        if vaa_data.vaa_version == 0
            || vaa_data.vaa_time == 0
            || &vaa_data.vaa_signature_account != accs.signature_set.key
        {
            return Err(SolitaireError::ProgramError(
                ProgramError::InvalidAccountData,
            ));
        }

        let submission_time = if vaa_data.submission_time == 0 {
            DEFAULT_SUBMISSION_TIME
        } else {
            vaa_data.submission_time
        };

        if (submission_time as i64) > accs.clock.unix_timestamp - RETENTION_PERIOD {
            return Err(MessageWithinRetentionWindow.into());
        }

        // Close PostedVAA: drain lamports, truncate data, hand back to system program.
        let vaa_lamports = accs.posted_vaa.lamports();
        **accs.fee_collector.lamports.borrow_mut() = accs
            .fee_collector
            .lamports()
            .checked_add(vaa_lamports)
            .ok_or(MathOverflow)?;
        **accs.posted_vaa.lamports.borrow_mut() = 0;
        accs.posted_vaa.realloc(0, false)?;
        accs.posted_vaa
            .assign(&solana_program::system_program::id());
    } else {
        // 5b. PostedVAA is NOT initialised: check guardian set is expired.
        // Verify the guardian set derivation matches the signature set's index.
        let gs_derivation = GuardianSetDerivationData {
            index: sig_data.guardian_set_index,
        };
        accs.guardian_set
            .verify_derivation(ctx.program_id, &gs_derivation)?;

        // The guardian set must NOT be active (i.e. it must be expired).
        if check_active(&accs.guardian_set, &accs.clock).is_ok() {
            return Err(GuardianSetStillActive.into());
        }
    }

    // 6. Close signature_set: drain lamports, truncate data, hand back to system program.
    let sig_lamports = accs.signature_set.lamports();
    **accs.fee_collector.lamports.borrow_mut() = accs
        .fee_collector
        .lamports()
        .checked_add(sig_lamports)
        .ok_or(MathOverflow)?;
    **accs.signature_set.lamports.borrow_mut() = 0;
    accs.signature_set.realloc(0, false)?;
    accs.signature_set
        .assign(&solana_program::system_program::id());

    // 7. Update bridge.last_lamports to prevent fee waiver.
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    Ok(())
}
