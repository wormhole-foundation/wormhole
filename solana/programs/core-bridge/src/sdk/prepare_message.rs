use anchor_lang::prelude::*;

#[cfg(feature = "cpi")]
#[derive(Accounts)]
pub struct PrepareMessage<'info> {
    /// Emitter authority (signer), who acts as the authority for crafting a Wormhole mesage.
    ///
    /// CHECK: This account is written to in the message that acts as the authority to continue
    /// writing to the message account. Once this authority is set, no one else can write and
    /// finalize this message.
    pub emitter_authority: AccountInfo<'info>,

    /// Core Bridge Message (mut).
    ///
    /// CHECK: This account can either be a generated keypair or an integrator's PDA. This message
    /// account can be closed only when it is in writing status.
    pub message: AccountInfo<'info>,
}

/// Trait for invoking the Core Bridge program's [init_message_v1](crate::cpi::init_message_v1),
/// [write_message_v1](crate::cpi::write_message_v1) and
/// [finalize_message_v1](crate::cpi::finalize_message_v1) instructions. These instructions are used
/// in concert with each other to prepare a message, which can be posted either within a program via
/// CPI or within the same transaction block as an instruction following your program's instruction.
// pub trait PrepareMessage<'info> {
//     fn core_bridge_program(&self) -> AccountInfo<'info>;

//     /// Core Bridge Emitter Authority (read-only signer). This emitter authority acts as the signer
//     /// for preparing a message before it is posted.
//     fn core_emitter_authority(&self) -> AccountInfo<'info>;

//     /// Core Bridge Message (mut).
//     fn core_message(&self) -> AccountInfo<'info>;
// }

/// SDK method for preparing a new Core Bridge message. It is assumed that the emitter authority is
/// your program's PDA, so emitter authority seeds are required to sign for these Core Bridge
/// Program instructions.
///
/// NOTE: When using this SDK method, be aware that the message account is not created yet. You must
/// either invoke [create_account](crate::sdk::cpi::system_program::create_account) or use Anchor's
/// `init` macro directive before calling this method.
#[cfg(feature = "cpi")]
pub fn prepare_message<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, PrepareMessage<'info>>,
    init_args: super::InitMessageV1Args,
    data: Vec<u8>,
) -> Result<()> {
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                draft_message: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        init_args,
    )?;

    crate::cpi::write_message_v1(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            crate::cpi::accounts::WriteMessageV1 {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                draft_message: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        crate::processor::WriteMessageV1Args { index: 0, data },
    )?;

    crate::cpi::finalize_message_v1(CpiContext::new_with_signer(
        ctx.program,
        crate::cpi::accounts::FinalizeMessageV1 {
            emitter_authority: ctx.accounts.emitter_authority,
            draft_message: ctx.accounts.message,
        },
        ctx.signer_seeds,
    ))
}

/// SDK method for initializing a new Core Bridge message by starting to write data to a message
/// account. If the message requires multiple calls, using this method may be convenient to begin
/// writing and then following this call with a subsequent [write_message] or
/// [write_and_finalize_message] call.
#[cfg(feature = "cpi")]
pub fn init_and_write_message<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, PrepareMessage<'info>>,
    init_args: super::InitMessageV1Args,
    index: u32,
    data: Vec<u8>,
) -> Result<()> {
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                draft_message: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        init_args,
    )?;

    crate::cpi::write_message_v1(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            crate::cpi::accounts::WriteMessageV1 {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                draft_message: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        crate::processor::WriteMessageV1Args { index, data },
    )
}

/// SDK method for writing and then finalizing an existing Core Bridge message if it is still in
/// [Writing](crate::state::MessageStatus::Writing) status. This method may be convenient to wrap up
/// writing data when it follows either [init_and_write_message] or [write_message].
#[cfg(feature = "cpi")]
pub fn write_and_finalize_message<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, PrepareMessage<'info>>,
    index: u32,
    data: Vec<u8>,
) -> Result<()> {
    crate::cpi::write_message_v1(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            crate::cpi::accounts::WriteMessageV1 {
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
                draft_message: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        crate::processor::WriteMessageV1Args { index, data },
    )?;

    crate::cpi::finalize_message_v1(CpiContext::new_with_signer(
        ctx.program,
        crate::cpi::accounts::FinalizeMessageV1 {
            emitter_authority: ctx.accounts.emitter_authority,
            draft_message: ctx.accounts.message,
        },
        ctx.signer_seeds,
    ))
}
