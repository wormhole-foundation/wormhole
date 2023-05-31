mod instruction;
mod processor;
pub mod state;
pub mod utils;

pub use crate::ID;
pub(crate) use processor::*;

#[cfg(feature = "cpi")]
pub mod cpi {
    pub use instruction::{LegacyPostMessageArgs, LegacyPostMessageUnreliableArgs};

    use crate::legacy::instruction::{PostMessage, PostMessageUnreliable};
    use anchor_lang::prelude::*;
    use solana_program::program::invoke_signed;

    use super::*;

    pub fn legacy_post_message<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, LegacyPostMessage<'info>>,
        args: LegacyPostMessageArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::post_message(
                PostMessage {
                    bridge: *ctx.accounts.bridge.key,
                    message: *ctx.accounts.message.key,
                    emitter: *ctx.accounts.emitter.key,
                    emitter_sequence: *ctx.accounts.emitter_sequence.key,
                    payer: *ctx.accounts.payer.key,
                    fee_collector: *ctx.accounts.fee_collector.key,
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
    pub fn legacy_post_message_unreliable<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, LegacyPostMessageUnreliable<'info>>,
        args: LegacyPostMessageUnreliableArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::post_message_unreliable(
                PostMessageUnreliable {
                    bridge: *ctx.accounts.bridge.key,
                    message: *ctx.accounts.message.key,
                    emitter: *ctx.accounts.emitter.key,
                    emitter_sequence: *ctx.accounts.emitter_sequence.key,
                    payer: *ctx.accounts.payer.key,
                    fee_collector: *ctx.accounts.fee_collector.key,
                    system_program: *ctx.accounts.system_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    #[derive(Accounts)]
    pub struct LegacyPostMessage<'info> {
        /// CHECK: Core Bridge Program Data (mut, seeds = ["bridge"]).
        pub bridge: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only signer).
        pub emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = ["Sequence", emitter.key]).
        pub emitter_sequence: AccountInfo<'info>,
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = ["fee_collector"]).
        pub fee_collector: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct LegacyPostMessageUnreliable<'info> {
        /// CHECK: Core Bridge Program Data (mut, seeds = \["bridge"\]).
        pub bridge: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only signer).
        pub emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
        pub emitter_sequence: AccountInfo<'info>,
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = \["fee_collector"\]).
        pub fee_collector: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
    }
}
