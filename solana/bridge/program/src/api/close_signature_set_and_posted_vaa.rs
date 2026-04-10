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

/// 30 days in seconds.
const RETENTION_PERIOD: i64 = 30 * 24 * 60 * 60;

/// Default submission_time for accounts created before the submission_time
/// field was populated. Gives legacy accounts a 30-day grace period from
/// this date (10 April 2026) instead of being immediately closable.
const DEFAULT_SUBMISSION_TIME: u32 = 1775833078;

#[derive(FromAccounts)]
pub struct CloseSignatureSetAndPostedVAA<'b> {
    /// Bridge config, used to update last_lamports.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// The signature set account to close.
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

    // 4. Determine if PostedVAA is initialised
    let vaa_initialized = accs.posted_vaa.data_len() > 0 && accs.posted_vaa.owner == ctx.program_id;

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

        let submission_time = if vaa_data.submission_time == 0 {
            DEFAULT_SUBMISSION_TIME
        } else {
            vaa_data.submission_time
        };

        if (submission_time as i64) > accs.clock.unix_timestamp - RETENTION_PERIOD {
            return Err(MessageWithinRetentionWindow.into());
        }

        // Close PostedVAA: transfer lamports to fee_collector.
        let vaa_lamports = accs.posted_vaa.lamports();
        **accs.fee_collector.lamports.borrow_mut() += vaa_lamports;
        **accs.posted_vaa.lamports.borrow_mut() = 0;
        accs.posted_vaa.data.borrow_mut().fill(0);
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

    // 6. Close signature_set: transfer lamports to fee_collector.
    let sig_lamports = accs.signature_set.lamports();
    **accs.fee_collector.lamports.borrow_mut() += sig_lamports;
    **accs.signature_set.lamports.borrow_mut() = 0;
    accs.signature_set.data.borrow_mut().fill(0);
    accs.signature_set
        .assign(&solana_program::system_program::id());

    // 7. Update bridge.last_lamports to prevent fee waiver.
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    Ok(())
}
