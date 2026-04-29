use crate::{
    accounts::{
        Bridge,
        FeeCollector,
    },
    error::Error::{
        InvalidAccountPrefix,
        InvalidProgramOwner,
        MathOverflow,
        MessageWithinRetentionWindow,
    },
    MessageData,
};
use solana_program::{
    program_error::ProgramError,
    sysvar::clock::Clock,
};
use solitaire::*;

use super::RETENTION_PERIOD;

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

        // Check retention: submission_time must be at or before (now - RETENTION_PERIOD).
        if (message_data.submission_time as i64) > accs.clock.unix_timestamp - RETENTION_PERIOD {
            return Err(MessageWithinRetentionWindow.into());
        }

        data.to_vec()
        // data borrow dropped here
    };

    // 5. Verify self_program (used as the CPI target) really is this program.
    if accs.self_program.key != ctx.program_id {
        return Err(InvalidProgramOwner.into());
    }

    // 6. Emit Anchor CPI event with the full account data (prefix + message).
    emit_event_cpi(
        ctx,
        &accs.event_authority,
        &MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR,
        &account_data,
    )?;

    // 7. Close the account: transfer lamports to fee_collector.
    let message_lamports = accs.message.lamports();
    **accs.fee_collector.lamports.borrow_mut() = accs
        .fee_collector
        .lamports()
        .checked_add(message_lamports)
        .ok_or(MathOverflow)?;
    **accs.message.lamports.borrow_mut() = 0;
    accs.message.data.borrow_mut().fill(0);
    accs.message.assign(&solana_program::system_program::id());

    // 8. Update bridge.last_lamports to prevent fee waiver.
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    Ok(())
}
