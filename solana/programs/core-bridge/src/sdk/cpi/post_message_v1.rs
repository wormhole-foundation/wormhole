pub use crate::legacy::cpi::PostMessageArgs;

use anchor_lang::prelude::*;

use super::InvokeCoreBridge;

pub trait InvokePostMessageV1<'info>: InvokeCoreBridge<'info> {
    fn payer(&self) -> AccountInfo<'info>;

    fn config(&self) -> AccountInfo<'info>;

    fn message(&self) -> AccountInfo<'info>;

    fn emitter(&self) -> AccountInfo<'info>;

    fn emitter_sequence(&self) -> AccountInfo<'info>;

    fn fee_collector(&self) -> Option<AccountInfo<'info>>;

    fn system_program(&self) -> AccountInfo<'info>;
}

pub fn post_prepared_message_v1<'info, A: InvokePostMessageV1<'info>>(
    accounts: &A,
    emitter_seeds: &[&[u8]],
    message_seeds: Option<&[&[u8]]>,
) -> Result<()> {
    post_new_message_v1_bytes(
        accounts,
        PostMessageArgs {
            nonce: Default::default(), // not checked
            payload: Vec::new(),
            commitment: crate::types::Commitment::Finalized, // not checked
        },
        emitter_seeds,
        message_seeds,
    )
}

pub struct PostNewMessageV1Args<W: wormhole_io::Writeable> {
    pub nonce: u32,
    pub message: W,
    pub commitment: crate::types::Commitment,
}

pub fn post_new_message_v1<'info, A: InvokePostMessageV1<'info>, W: wormhole_io::Writeable>(
    accounts: &A,
    args: PostNewMessageV1Args<W>,
    emitter_seeds: &[&[u8]],
    message_seeds: Option<&[&[u8]]>,
) -> Result<()> {
    post_new_message_v1_bytes(
        accounts,
        PostMessageArgs {
            nonce: args.nonce,
            payload: args.message.to_vec(),
            commitment: args.commitment,
        },
        emitter_seeds,
        message_seeds,
    )
}

pub fn post_new_message_v1_bytes<'info, A: InvokePostMessageV1<'info>>(
    accounts: &A,
    args: crate::legacy::cpi::PostMessageArgs,
    emitter_seeds: &[&[u8]],
    message_seeds: Option<&[&[u8]]>,
) -> Result<()> {
    // If there is a fee, transfer it. But only try if the fee collector is provided because the
    // post message instruction will fail if there is actually a fee but no fee collector.
    if let Some(fee_collector) = accounts.fee_collector() {
        let fee_lamports =
            crate::zero_copy::Config::parse(&accounts.config().try_borrow_data()?)?.fee_lamports();

        if fee_lamports > 0 {
            anchor_lang::system_program::transfer(
                CpiContext::new(
                    accounts.system_program(),
                    anchor_lang::system_program::Transfer {
                        from: accounts.payer(),
                        to: fee_collector,
                    },
                ),
                fee_lamports,
            )?;
        }
    }

    match message_seeds {
        Some(message_seeds) => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.config(),
                    message: accounts.message(),
                    emitter: accounts.emitter(),
                    emitter_sequence: accounts.emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.fee_collector(),
                    system_program: accounts.system_program(),
                },
                &[emitter_seeds, message_seeds],
            ),
            args,
        ),
        None => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.config(),
                    message: accounts.message(),
                    emitter: accounts.emitter(),
                    emitter_sequence: accounts.emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.fee_collector(),
                    system_program: accounts.system_program(),
                },
                &[emitter_seeds],
            ),
            args,
        ),
    }
}
