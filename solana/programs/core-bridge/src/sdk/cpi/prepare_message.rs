pub use crate::processor::InitMessageV1Args;

use anchor_lang::prelude::*;

/// Trait for invoking the Core Bridge program's [init_message_v1](crate::cpi::init_message_v1),
/// [write_message_v1](crate::cpi::write_message_v1) and
/// [finalize_message_v1](crate::cpi::finalize_message_v1) instructions. These instructions are used
/// in concert with each other to prepare a message, which can be posted either within a program via
/// CPI or within the same transaction block as an instruction following your program's instruction.
pub trait PrepareMessage<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info>;

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
/// either invoke [create_account](crate::sdk::cpi::system_program::create_account) or use Anchor's
/// `init` macro directive before calling this method.
pub fn prepare_message<'info, A>(
    accounts: &A,
    init_args: InitMessageV1Args,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PrepareMessage<'info>,
{
    handle_prepare_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.core_emitter_authority(),
        init_args,
        data,
        signer_seeds,
    )
}

/// SDK method for initializing a new Core Bridge message by starting to write data to a message
/// account. If the message requires multiple calls, using this method may be convenient to begin
/// writing and then following this call with a subsequent [write_message] or
/// [write_and_finalize_message] call.
pub fn init_and_write_message<'info, A>(
    accounts: &A,
    init_args: InitMessageV1Args,
    index: u32,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PrepareMessage<'info>,
{
    handle_init_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.core_emitter_authority(),
        init_args,
        signer_seeds,
    )?;

    write_message(accounts, index, data, signer_seeds)
}

/// SDK method for writing to an existing Core Bridge message if it is still in
/// [Writing](crate::state::MessageStatus::Writing) status.
pub fn write_message<'info, A>(
    accounts: &A,
    index: u32,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PrepareMessage<'info>,
{
    handle_write_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.core_emitter_authority(),
        index,
        data,
        signer_seeds,
    )
}

/// SDK method for writing and then finalizing an existing Core Bridge message if it is still in
/// [Writing](crate::state::MessageStatus::Writing) status. This method may be convenient to wrap up
/// writing data when it follows either [init_and_write_message] or [write_message].
pub fn write_and_finalize_message<'info, A>(
    accounts: &A,
    index: u32,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PrepareMessage<'info>,
{
    write_message(accounts, index, data, signer_seeds)?;

    handle_finalize_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.core_emitter_authority(),
        signer_seeds,
    )
}

pub(in crate::sdk::cpi) fn handle_prepare_message_v1<'info>(
    core_bridge_program: AccountInfo<'info>,
    core_message: AccountInfo<'info>,
    core_emitter_authority: AccountInfo<'info>,
    init_args: InitMessageV1Args,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()> {
    // Initialize message with created account.
    handle_init_message_v1(
        core_bridge_program.clone(),
        core_message.clone(),
        core_emitter_authority.clone(),
        init_args,
        signer_seeds,
    )?;

    handle_write_message_v1(
        core_bridge_program.clone(),
        core_message.clone(),
        core_emitter_authority.clone(),
        0,
        data,
        signer_seeds,
    )?;

    handle_finalize_message_v1(
        core_bridge_program.clone(),
        core_message.clone(),
        core_emitter_authority.clone(),
        signer_seeds,
    )
}

fn handle_init_message_v1<'info>(
    core_bridge_program: AccountInfo<'info>,
    core_message: AccountInfo<'info>,
    core_emitter_authority: AccountInfo<'info>,
    init_args: InitMessageV1Args,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()> {
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            core_bridge_program.clone(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: core_emitter_authority.clone(),
                draft_message: core_message.clone(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        init_args,
    )
}

fn handle_write_message_v1<'info>(
    core_bridge_program: AccountInfo<'info>,
    core_message: AccountInfo<'info>,
    core_emitter_authority: AccountInfo<'info>,
    index: u32,
    data: Vec<u8>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()> {
    crate::cpi::write_message_v1(
        CpiContext::new_with_signer(
            core_bridge_program.clone(),
            crate::cpi::accounts::WriteMessageV1 {
                emitter_authority: core_emitter_authority.clone(),
                draft_message: core_message.clone(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        crate::processor::WriteMessageV1Args { index, data },
    )
}

fn handle_finalize_message_v1<'info>(
    core_bridge_program: AccountInfo<'info>,
    core_message: AccountInfo<'info>,
    core_emitter_authority: AccountInfo<'info>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()> {
    crate::cpi::finalize_message_v1(CpiContext::new_with_signer(
        core_bridge_program.clone(),
        crate::cpi::accounts::FinalizeMessageV1 {
            emitter_authority: core_emitter_authority.clone(),
            draft_message: core_message.clone(),
        },
        signer_seeds.unwrap_or_default(),
    ))
}
