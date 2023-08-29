mod instruction;
mod processor;
pub mod state;

pub use crate::ID;
pub(crate) use instruction::*;
pub(crate) use processor::*;

#[cfg(feature = "cpi")]
pub mod cpi {
    pub use instruction::{EmptyArgs, TransferTokensArgs, TransferTokensWithPayloadArgs};

    use anchor_lang::prelude::*;
    use solana_program::program::invoke_signed;

    use super::*;

    pub fn transfer_tokens_native<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, TransferTokensNative<'info>>,
        args: TransferTokensArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_native(
                instruction::TransferTokensNative {
                    payer: *ctx.accounts.payer.key,
                    src_token: *ctx.accounts.src_token.key,
                    mint: *ctx.accounts.mint.key,
                    custody_token: *ctx.accounts.custody_token.key,
                    transfer_authority: *ctx.accounts.transfer_authority.key,
                    custody_authority: *ctx.accounts.custody_authority.key,
                    core_bridge_config: *ctx.accounts.core_bridge_config.key,
                    core_message: *ctx.accounts.core_message.key,
                    core_emitter: *ctx.accounts.core_emitter.key,
                    core_emitter_sequence: *ctx.accounts.core_emitter_sequence.key,
                    core_fee_collector: ctx
                        .accounts
                        .core_fee_collector
                        .as_ref()
                        .map(|info| *info.key),
                    system_program: *ctx.accounts.system_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn transfer_tokens_wrapped<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, TransferTokensWrapped<'info>>,
        args: TransferTokensArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_wrapped(
                instruction::TransferTokensWrapped {
                    payer: *ctx.accounts.payer.key,
                    src_token: *ctx.accounts.src_token.key,
                    wrapped_mint: *ctx.accounts.wrapped_mint.key,
                    wrapped_asset: *ctx.accounts.wrapped_asset.key,
                    transfer_authority: *ctx.accounts.transfer_authority.key,
                    core_bridge_config: *ctx.accounts.core_bridge_config.key,
                    core_message: *ctx.accounts.core_message.key,
                    core_emitter: *ctx.accounts.core_emitter.key,
                    core_emitter_sequence: *ctx.accounts.core_emitter_sequence.key,
                    core_fee_collector: ctx
                        .accounts
                        .core_fee_collector
                        .as_ref()
                        .map(|info| *info.key),
                    system_program: *ctx.accounts.system_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn transfer_tokens_with_payload_native<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, TransferTokensWithPayloadNative<'info>>,
        args: TransferTokensWithPayloadArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_with_payload_native(
                instruction::TransferTokensWithPayloadNative {
                    payer: *ctx.accounts.payer.key,
                    src_token: *ctx.accounts.src_token.key,
                    mint: *ctx.accounts.mint.key,
                    custody_token: *ctx.accounts.custody_token.key,
                    transfer_authority: *ctx.accounts.transfer_authority.key,
                    custody_authority: *ctx.accounts.custody_authority.key,
                    core_bridge_config: *ctx.accounts.core_bridge_config.key,
                    core_message: *ctx.accounts.core_message.key,
                    core_emitter: *ctx.accounts.core_emitter.key,
                    core_emitter_sequence: *ctx.accounts.core_emitter_sequence.key,
                    core_fee_collector: ctx
                        .accounts
                        .core_fee_collector
                        .as_ref()
                        .map(|info| *info.key),
                    sender_authority: *ctx.accounts.sender_authority.key,
                    system_program: *ctx.accounts.system_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn transfer_tokens_with_payload_wrapped<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, TransferTokensWithPayloadWrapped<'info>>,
        args: TransferTokensWithPayloadArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_with_payload_wrapped(
                instruction::TransferTokensWithPayloadWrapped {
                    payer: *ctx.accounts.payer.key,
                    src_token: *ctx.accounts.src_token.key,
                    wrapped_mint: *ctx.accounts.wrapped_mint.key,
                    wrapped_asset: *ctx.accounts.wrapped_asset.key,
                    transfer_authority: *ctx.accounts.transfer_authority.key,
                    core_bridge_config: *ctx.accounts.core_bridge_config.key,
                    core_message: *ctx.accounts.core_message.key,
                    core_emitter: *ctx.accounts.core_emitter.key,
                    core_emitter_sequence: *ctx.accounts.core_emitter_sequence.key,
                    core_fee_collector: ctx
                        .accounts
                        .core_fee_collector
                        .as_ref()
                        .map(|info| *info.key),
                    sender_authority: *ctx.accounts.sender_authority.key,
                    system_program: *ctx.accounts.system_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    #[derive(Accounts)]
    pub struct TransferTokensNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Transfer Authority (mut, seeds = [mint.key], seeds::program =
        /// token_bridge_program).
        pub custody_token: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = ["authority_signer"], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Custody Authority (read-only, seeds = ["custody_signer"], seeds::program =
        /// token_bridge_program).
        pub custody_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = ["Bridge"], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = ["emitter"], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = ["Sequence", emitter.key],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = ["fee_collector"], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct TransferTokensWrapped<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Wrapped Mint (mut, seeds = ["wrapped", token_chain, token_address],
        /// seeds::program = token_bridge_program).
        pub wrapped_mint: AccountInfo<'info>,
        /// CHECK: Wrapped Asset (read-only, seeds = [wrapped_mint.key], seeds::program =
        /// token_bridge_program).
        pub wrapped_asset: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = ["authority_signer"], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = ["Bridge"], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = ["emitter"], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = ["Sequence", emitter.key],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = ["fee_collector"], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct TransferTokensWithPayloadNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Transfer Authority (mut, seeds = [mint.key], seeds::program =
        /// token_bridge_program).
        pub custody_token: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = ["authority_signer"], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Custody Authority (read-only, seeds = ["custody_signer"], seeds::program =
        /// token_bridge_program).
        pub custody_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = ["Bridge"], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = ["emitter"], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = ["Sequence", emitter.key],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = ["fee_collector"], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: Sender Authority (read-only, seeds = ["sender"]).
        pub sender_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct TransferTokensWithPayloadWrapped<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Wrapped Mint (mut, seeds = ["wrapped", token_chain, token_address],
        /// seeds::program = token_bridge_program).
        pub wrapped_mint: AccountInfo<'info>,
        /// CHECK: Wrapped Asset (read-only, seeds = [wrapped_mint.key], seeds::program =
        /// token_bridge_program).
        pub wrapped_asset: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = ["authority_signer"], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = ["Bridge"], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = ["emitter"], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = ["Sequence", emitter.key],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = ["fee_collector"], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: Sender Authority (read-only, seeds = ["sender"]).
        pub sender_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }
}
