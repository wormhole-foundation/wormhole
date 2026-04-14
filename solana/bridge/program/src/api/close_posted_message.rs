use crate::{
    accounts::{
        Bridge,
        FeeCollector,
    },
    error::Error::{
        InvalidAccountPrefix,
        InvalidEventAuthority,
        InvalidProgramOwner,
        MessageWithinRetentionWindow,
    },
    MessageData,
};
use solana_program::{
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::invoke_signed,
    program_error::ProgramError,
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::{
    EVENT_AUTHORITY_SEED,
    EVENT_IX_TAG_LE,
    *,
};

/// 30 days in seconds.
const RETENTION_PERIOD: i64 = 30 * 24 * 60 * 60;

/// Event discriminator: SHA256("event:MessageAccountClosed")[0..8].
pub const MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR: [u8; 8] =
    [0x9e, 0xf6, 0x1b, 0xc2, 0x24, 0x14, 0x28, 0xb9];

#[derive(FromAccounts)]
pub struct ClosePostedMessage<'b> {
    /// Bridge config, used to update last_lamports.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// The posted message account to close. Must have "msg" or "msu" prefix.
    /// Uses Info to avoid Persist serializing back into a zeroed account.
    pub message: Mut<Info<'b>>,

    /// Fee collector PDA to receive the reclaimed lamports.
    pub fee_collector: Mut<FeeCollector<'b>>,

    /// Clock for timestamp validation.
    pub clock: Sysvar<'b, Clock>,

    /// Event authority PDA for Anchor CPI event signing.
    pub event_authority: Info<'b>,

    /// This program (for self-CPI).
    pub self_program: Info<'b>,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct ClosePostedMessageData {}

pub fn close_posted_message(
    ctx: &ExecutionContext,
    accs: &mut ClosePostedMessage,
    _data: ClosePostedMessageData,
) -> Result<()> {
    // 1. Verify account is owned by this program.
    if accs.message.owner != ctx.program_id {
        return Err(InvalidProgramOwner.into());
    }

    // 2-4. Parse and validate the message, capturing full account data for the event.
    let account_data = {
        let data = accs.message.data.borrow();
        if data.len() < 3 {
            return Err(InvalidAccountPrefix.into());
        }

        let prefix = &data[0..3];
        if prefix != b"msg" && prefix != b"msu" {
            return Err(InvalidAccountPrefix.into());
        }

        let message_data = MessageData::try_from_slice(&data[3..])
            .map_err(|_| SolitaireError::ProgramError(ProgramError::InvalidAccountData))?;

        // Check retention: submission_time must be at or before (now - 30 days).
        if (message_data.submission_time as i64) > accs.clock.unix_timestamp - RETENTION_PERIOD {
            return Err(MessageWithinRetentionWindow.into());
        }

        data.to_vec()
        // data borrow dropped here
    };

    // 5. Emit Anchor CPI event with the full account data (prefix + message).
    emit_message_event(
        ctx,
        &accs.event_authority,
        &accs.self_program,
        &account_data,
    )?;

    // 6. Close the account: transfer lamports to fee_collector.
    let message_lamports = accs.message.lamports();
    **accs.fee_collector.lamports.borrow_mut() += message_lamports;
    **accs.message.lamports.borrow_mut() = 0;
    accs.message.data.borrow_mut().fill(0);
    accs.message.assign(&solana_program::system_program::id());

    // 7. Update bridge.last_lamports to prevent fee waiver.
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    Ok(())
}

/// Emit an Anchor-compliant CPI event containing the full account data.
fn emit_message_event(
    ctx: &ExecutionContext,
    event_authority: &Info,
    self_program: &Info,
    account_data: &[u8],
) -> Result<()> {
    // Verify self_program is actually our program.
    if self_program.key != ctx.program_id {
        return Err(InvalidEventAuthority.into());
    }

    // Compute event authority PDA and verify.
    let (expected_authority, bump) =
        Pubkey::find_program_address(&[EVENT_AUTHORITY_SEED], ctx.program_id);
    if event_authority.key != &expected_authority {
        return Err(InvalidEventAuthority.into());
    }

    // Build instruction data: selector + discriminator + raw account data.
    let mut ix_data = Vec::with_capacity(8 + 8 + account_data.len());
    ix_data.extend_from_slice(&EVENT_IX_TAG_LE);
    ix_data.extend_from_slice(&MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR);
    ix_data.extend_from_slice(account_data);

    let ix = Instruction {
        program_id: *ctx.program_id,
        accounts: vec![AccountMeta::new_readonly(expected_authority, true)],
        data: ix_data,
    };

    invoke_signed(&ix, ctx.accounts, &[&[EVENT_AUTHORITY_SEED, &[bump]]])?;

    Ok(())
}
