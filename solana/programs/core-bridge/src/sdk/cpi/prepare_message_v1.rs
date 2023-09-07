pub use crate::processor::InitMessageV1Args;

use anchor_lang::prelude::*;

use super::InvokeCoreBridge;

/// Trait for invoking the Core Bridge program's `init_message_v1` and `process_message_v1`
/// instructions. These instructions are used in concert with each other to prepare a message, which
/// can be posted either within a program via CPI or within the same transaction block as an
/// instruction following your program's instruction.
pub trait PrepareMessageV1<'info>: InvokeCoreBridge<'info> {
    /// Core Bridge Emitter Authority (read-only signer). This emitter authority acts as the signer
    /// for preparing a message before it is posted.
    fn core_emitter_authority(&self) -> AccountInfo<'info>;

    /// Core Bridge Message (mut).
    fn core_message(&self) -> AccountInfo<'info>;
}

/// SDK method for preparing a new Core Bridge message. It is assumed that the emitter authority is
/// your program's PDA, so emitter authority seeds are required to sign for these Core Bridge
/// Program instructions.
///
/// NOTE: When using this SDK method, be aware that the message account is not created yet. You must
/// invoke `system_program::create_account` before calling this method either using Anchor's `init`
/// macro directive or via System Program CPI.
pub fn prepare_message_v1<'info, A>(
    accounts: &A,
    init_args: InitMessageV1Args,
    data: Vec<u8>,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()>
where
    A: PrepareMessageV1<'info>,
{
    handle_prepare_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.core_emitter_authority(),
        init_args,
        data,
        emitter_authority_seeds,
    )
}

/// SDK method for initializing a new Core Bridge message by starting to write data to a message
/// account. If the message requires multiple calls, using this method may be convenient to begin
/// writing and then following this call with a subsequent [write_message_v1] or
/// [write_and_finalize_message_v1] call.
pub fn init_and_write_message_v1<'info, A>(
    accounts: &A,
    args: InitMessageV1Args,
    index: u32,
    data: Vec<u8>,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()>
where
    A: PrepareMessageV1<'info>,
{
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: accounts.core_emitter_authority(),
                draft_message: accounts.core_message(),
            },
            &[emitter_authority_seeds],
        ),
        args,
    )?;

    write_message_v1(accounts, index, data, emitter_authority_seeds)
}

/// SDK method for writing to an existing Core Bridge message if it is still in
/// [Writing](crate::state::MessageStatus::Writing) status.
pub fn write_message_v1<'info, A>(
    accounts: &A,
    index: u32,
    data: Vec<u8>,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()>
where
    A: PrepareMessageV1<'info>,
{
    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: accounts.core_emitter_authority(),
                draft_message: accounts.core_message(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Write { index, data },
    )
}

/// SDK method for writing and then finalizing an existing Core Bridge message if it is still in
/// [Writing](crate::state::MessageStatus::Writing) status. This method may be convenient to wrap up
/// writing data when it follows either [init_and_write_message_v1] or [write_message_v1].
pub fn write_and_finalize_message_v1<'info, A>(
    accounts: &A,
    index: u32,
    data: Vec<u8>,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()>
where
    A: PrepareMessageV1<'info>,
{
    write_message_v1(accounts, index, data, emitter_authority_seeds)?;

    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: accounts.core_emitter_authority(),
                draft_message: accounts.core_message(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Finalize,
    )
}

pub(in crate::sdk) fn handle_prepare_message_v1<'info>(
    core_bridge_program: AccountInfo<'info>,
    core_message: AccountInfo<'info>,
    core_emitter_authority: AccountInfo<'info>,
    init_args: InitMessageV1Args,
    data: Vec<u8>,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()> {
    // Initialize message with created account.
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            core_bridge_program.clone(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: core_emitter_authority.clone(),
                draft_message: core_message.clone(),
            },
            &[emitter_authority_seeds],
        ),
        init_args,
    )?;

    // Write message.
    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            core_bridge_program.clone(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: core_emitter_authority.clone(),
                draft_message: core_message.clone(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Write { index: 0, data },
    )?;

    // Finalize.
    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            core_bridge_program.clone(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: core_emitter_authority.clone(),
                draft_message: core_message.clone(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Finalize,
    )
}
