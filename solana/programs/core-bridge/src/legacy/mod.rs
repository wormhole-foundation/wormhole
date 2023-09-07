//! Legacy Core Bridge state and instruction processing.

pub use crate::ID;

pub mod accounts;

pub mod instruction;

mod processor;
pub(crate) use processor::*;

pub mod state;

pub mod utils;

/// Collection of methods to interact with the Core Bridge program via CPI. The structs defined in
/// this module mirror the structs deriving [Accounts](anchor_lang::prelude::Accounts), where each
/// field is an [AccountInfo]. **Integrators: Please use [sdk](crate::sdk) instead of this module.**
///
/// NOTE: This is similar to how [cpi](mod@crate::cpi) is generated via Anchor's
/// [program][anchor_lang::prelude::program] macro.
#[cfg(feature = "cpi")]
pub mod cpi {
    use anchor_lang::prelude::*;
    use solana_program::program::invoke_signed;

    use super::*;

    /// Processor to post (publish) a Wormhole message by setting up the message account for
    /// Guardian observation.
    ///
    /// A message is either created beforehand using the new Anchor instruction to process a message
    /// or is created at this point.
    pub fn post_message<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, PostMessage<'info>>,
        args: instruction::PostMessageArgs,
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

    /// Processor to post (publish) a Wormhole message by setting up the message account for
    /// Guardian observation. This message account has either been created already or is created in
    /// this call.
    ///
    /// If this message account already exists, the emitter must be the same as the one encoded in
    /// the message and the payload must be the same size.
    pub fn post_message_unreliable<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, PostMessageUnreliable<'info>>,
        args: instruction::PostMessageArgs,
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
        /// CHECK: Core Bridge Fee Collector (optional, read-only, seeds = \["fee_collector"\]).
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
        /// CHECK: Core Bridge Fee Collector (optional, read-only, seeds = \["fee_collector"\]).
        pub fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
    }
}
