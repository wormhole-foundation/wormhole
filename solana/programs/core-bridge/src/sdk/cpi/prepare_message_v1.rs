pub use crate::processor::InitMessageV1Args;

use anchor_lang::prelude::*;

use super::InvokeCoreBridge;

pub trait InvokePrepareMessageV1<'info>: InvokeCoreBridge<'info> {
    fn emitter_authority(&self) -> AccountInfo<'info>;

    fn message(&self) -> AccountInfo<'info>;
}

pub fn prepare_message_v1<'info, A: InvokePrepareMessageV1<'info>, W: wormhole_io::Writeable>(
    accounts: &A,
    message: W,
    init_args: InitMessageV1Args,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()> {
    prepare_message_v1_bytes(
        accounts,
        message.to_vec(),
        init_args,
        emitter_authority_seeds,
    )
}

pub fn prepare_message_v1_bytes<'info, A: InvokePrepareMessageV1<'info>>(
    accounts: &A,
    data: Vec<u8>,
    init_args: InitMessageV1Args,
    emitter_authority_seeds: &[&[u8]],
) -> Result<()> {
    // Initialize message with created account.
    crate::cpi::init_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::InitMessageV1 {
                emitter_authority: accounts.emitter_authority(),
                draft_message: accounts.message(),
            },
            &[emitter_authority_seeds],
        ),
        init_args,
    )?;

    // Write message.
    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: accounts.emitter_authority(),
                draft_message: accounts.message(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Write { index: 0, data },
    )?;

    // Finalize.
    crate::cpi::process_message_v1(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::cpi::accounts::ProcessMessageV1 {
                emitter_authority: accounts.emitter_authority(),
                draft_message: accounts.message(),
                close_account_destination: None,
            },
            &[emitter_authority_seeds],
        ),
        crate::processor::ProcessMessageV1Directive::Finalize,
    )
}
