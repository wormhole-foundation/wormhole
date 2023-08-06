//! Legacy Core Bridge state and instruction processing.

pub use crate::ID;

pub mod accounts;

pub mod instruction;

mod processor;
pub(crate) use processor::*;

pub mod state;

pub mod utils;

/// **Integrators: Please use [sdk](mod@crate::sdk) instead of this module.** Methods used to
/// interact with the Core Bridge Program via CPI.
#[cfg(feature = "cpi")]
pub mod cpi {
    pub use instruction::PostMessageArgs;

    use anchor_lang::prelude::*;
    use solana_program::program::invoke_signed;

    use super::*;

    pub fn post_message<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, PostMessage<'info>>,
        args: PostMessageArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::post_message(
                crate::legacy::accounts::PostMessage {
                    config: *ctx.accounts.config.key,
                    message: *ctx.accounts.message.key,
                    emitter: ctx.accounts.emitter.as_ref().map(|info| *info.key),
                    emitter_sequence: *ctx.accounts.emitter_sequence.key,
                    payer: *ctx.accounts.payer.key,
                    fee_collector: ctx.accounts.fee_collector.as_ref().map(|info| *info.key),
                    system_program: *ctx.accounts.system_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    /// This instruction handler is used to post a new message to the core bridge using an existing
    /// message account.
    ///
    /// The constraints for posting a message using this instruction handler are:
    /// * Emitter must be the same as the message account's emitter.
    /// * The new message must be the same size as the existing message's payload.
    pub fn post_message_unreliable<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, PostMessageUnreliable<'info>>,
        args: PostMessageArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::post_message_unreliable(
                crate::legacy::accounts::PostMessageUnreliable {
                    config: *ctx.accounts.config.key,
                    message: *ctx.accounts.message.key,
                    emitter: *ctx.accounts.emitter.key,
                    emitter_sequence: *ctx.accounts.emitter_sequence.key,
                    payer: *ctx.accounts.payer.key,
                    fee_collector: ctx.accounts.fee_collector.as_ref().map(|info| *info.key),
                    system_program: *ctx.accounts.system_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    /// Context to post a new Core Bridge message.
    #[derive(Accounts)]
    pub struct PostMessage<'info> {
        /// CHECK: Core Bridge Program Data (mut, seeds = \["Bridge"\]).
        pub config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (optional, read-only signer).
        pub emitter: Option<AccountInfo<'info>>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
        pub emitter_sequence: AccountInfo<'info>,
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (optional, mut, seeds = \["fee_collector"\]).
        pub fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
    }

    /// Context to post a new or reuse an existing Core Bridge message.
    #[derive(Accounts)]
    pub struct PostMessageUnreliable<'info> {
        /// CHECK: Core Bridge Program Data (mut, seeds = \["Bridge"\]).
        pub config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only signer).
        pub emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
        pub emitter_sequence: AccountInfo<'info>,
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (optional, mut, seeds = \["fee_collector"\]).
        pub fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
    }
}
