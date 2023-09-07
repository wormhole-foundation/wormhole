//! Legacy Token Bridge state and instruction processing.

pub use crate::ID;

pub mod accounts;

pub mod instruction;

mod processor;
pub(crate) use processor::*;

pub mod state;

/// Collection of methods to interact with the Token Bridge program via CPI. The structs defined in
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

    pub fn complete_transfer_native<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, CompleteTransferNative<'info>>,
    ) -> Result<()> {
        invoke_signed(
            &instruction::complete_transfer_native(accounts::CompleteTransferNative {
                payer: *ctx.accounts.payer.key,
                vaa: *ctx.accounts.vaa.key,
                claim: *ctx.accounts.claim.key,
                registered_emitter: *ctx.accounts.registered_emitter.key,
                recipient_token: *ctx.accounts.recipient_token.key,
                payer_token: *ctx.accounts.payer_token.key,
                custody_token: *ctx.accounts.custody_token.key,
                mint: *ctx.accounts.mint.key,
                custody_authority: *ctx.accounts.custody_authority.key,
                recipient: ctx.accounts.recipient.as_ref().map(|info| *info.key),
                system_program: *ctx.accounts.system_program.key,
                token_program: *ctx.accounts.token_program.key,
            }),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn complete_transfer_wrapped<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, CompleteTransferWrapped<'info>>,
    ) -> Result<()> {
        invoke_signed(
            &instruction::complete_transfer_wrapped(accounts::CompleteTransferWrapped {
                payer: *ctx.accounts.payer.key,
                vaa: *ctx.accounts.vaa.key,
                claim: *ctx.accounts.claim.key,
                registered_emitter: *ctx.accounts.registered_emitter.key,
                recipient_token: *ctx.accounts.recipient_token.key,
                payer_token: *ctx.accounts.payer_token.key,
                wrapped_mint: *ctx.accounts.wrapped_mint.key,
                wrapped_asset: *ctx.accounts.wrapped_asset.key,
                mint_authority: *ctx.accounts.mint_authority.key,
                recipient: ctx.accounts.recipient.as_ref().map(|info| *info.key),
                system_program: *ctx.accounts.system_program.key,
                token_program: *ctx.accounts.token_program.key,
            }),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn complete_transfer_with_payload_native<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, CompleteTransferWithPayloadNative<'info>>,
    ) -> Result<()> {
        invoke_signed(
            &instruction::complete_transfer_with_payload_native(
                accounts::CompleteTransferWithPayloadNative {
                    payer: *ctx.accounts.payer.key,
                    vaa: *ctx.accounts.vaa.key,
                    claim: *ctx.accounts.claim.key,
                    registered_emitter: *ctx.accounts.registered_emitter.key,
                    dst_token: *ctx.accounts.dst_token.key,
                    redeemer_authority: *ctx.accounts.redeemer_authority.key,
                    custody_token: *ctx.accounts.custody_token.key,
                    mint: *ctx.accounts.mint.key,
                    custody_authority: *ctx.accounts.custody_authority.key,
                    system_program: *ctx.accounts.system_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn complete_transfer_with_payload_wrapped<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, CompleteTransferWithPayloadWrapped<'info>>,
    ) -> Result<()> {
        invoke_signed(
            &instruction::complete_transfer_with_payload_wrapped(
                accounts::CompleteTransferWithPayloadWrapped {
                    payer: *ctx.accounts.payer.key,
                    vaa: *ctx.accounts.vaa.key,
                    claim: *ctx.accounts.claim.key,
                    registered_emitter: *ctx.accounts.registered_emitter.key,
                    dst_token: *ctx.accounts.dst_token.key,
                    redeemer_authority: *ctx.accounts.redeemer_authority.key,
                    wrapped_mint: *ctx.accounts.wrapped_mint.key,
                    wrapped_asset: *ctx.accounts.wrapped_asset.key,
                    mint_authority: *ctx.accounts.mint_authority.key,
                    system_program: *ctx.accounts.system_program.key,
                    token_program: *ctx.accounts.token_program.key,
                },
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    pub fn transfer_tokens_native<'info>(
        ctx: CpiContext<'_, '_, '_, 'info, TransferTokensNative<'info>>,
        args: instruction::TransferTokensArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_native(
                accounts::TransferTokensNative {
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
                    token_program: *ctx.accounts.token_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
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
        args: instruction::TransferTokensArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_wrapped(
                accounts::TransferTokensWrapped {
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
                    token_program: *ctx.accounts.token_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
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
        args: instruction::TransferTokensWithPayloadArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_with_payload_native(
                accounts::TransferTokensWithPayloadNative {
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
                    token_program: *ctx.accounts.token_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
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
        args: instruction::TransferTokensWithPayloadArgs,
    ) -> Result<()> {
        invoke_signed(
            &instruction::transfer_tokens_with_payload_wrapped(
                accounts::TransferTokensWithPayloadWrapped {
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
                    token_program: *ctx.accounts.token_program.key,
                    core_bridge_program: *ctx.accounts.core_bridge_program.key,
                },
                args,
            ),
            &ctx.to_account_infos(),
            ctx.signer_seeds,
        )
        .map_err(Into::into)
    }

    #[derive(Accounts)]
    pub struct CompleteTransferNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Posted VAA Account (read-only, seeds = ["PostedVAA", message_hash],
        /// seeds::program = core_bridge_program).
        pub vaa: AccountInfo<'info>,
        /// CHECK: Claim Account (mut, seeds = [emitter_address, emitter_chain, sequence],
        /// seeds::program = token_bridge_program).
        pub claim: AccountInfo<'info>,
        /// CHECK: Registered Emitter Account (mut, seeds = \[emitter_chain\], seeds::program =
        /// token_bridge_program).
        ///
        /// NOTE: If the above PDA does not exist, there is a legacy account whose address is
        /// derived using seeds = \[emitter_chain, emitter_address\].
        pub registered_emitter: AccountInfo<'info>,
        /// CHECK: Recipient Token Account (mut).
        pub recipient_token: AccountInfo<'info>,
        /// CHECK: Payer (Relayer) Token Account (mut).
        pub payer_token: AccountInfo<'info>,
        /// CHECK: Custody Token Account (mut, seeds = \[mint.key\], seeds::program =
        /// token_bridge_program).
        pub custody_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Custody Authority (read-only, seeds = \["custody_signer"\], seeds::program =
        /// token_bridge_program).
        pub custody_authority: AccountInfo<'info>,
        /// CHECK: Recipient, which should be the account owner of recipient token (read-only).
        ///
        /// NOTE: This used to be the rent sysvar. If the VAA encodes the recipient token account,
        /// this account does not need to be provided. Otherwise you need to provide this account,
        /// whose pubkey should match the VAA recipient.
        pub recipient: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct CompleteTransferWrapped<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Posted VAA Account (read-only, seeds = \["PostedVAA", message_hash\],
        /// seeds::program = core_bridge_program).
        pub vaa: AccountInfo<'info>,
        /// CHECK: Claim Account (mut, seeds = \[emitter_address, emitter_chain, sequence\],
        /// seeds::program = token_bridge_program).
        pub claim: AccountInfo<'info>,
        /// CHECK: Registered Emitter Account (mut, seeds = \[emitter_chain\], seeds::program =
        /// token_bridge_program).
        ///
        /// NOTE: If the above PDA does not exist, there is a legacy account whose address is
        /// derived using seeds = \[emitter_chain, emitter_address\].
        pub registered_emitter: AccountInfo<'info>,
        /// CHECK: Recipient Token Account (mut).
        pub recipient_token: AccountInfo<'info>,
        /// CHECK: Payer (Relayer) Token Account (mut).
        pub payer_token: AccountInfo<'info>,
        /// CHECK: Wrapped Mint (mut, seeds = \["wrapped", token_chain, token_address\],
        /// seeds::program = token_bridge_program).
        pub wrapped_mint: AccountInfo<'info>,
        /// CHECK: Wrapped Asset (read-only, seeds = \["meta", wrapped_mint.key\], seeds::program =
        /// token_bridge_program).
        pub wrapped_asset: AccountInfo<'info>,
        /// CHECK: Mint Authority (read-only, seeds = \["mint_signer"\], seeds::program =
        /// token_bridge_program).
        pub mint_authority: AccountInfo<'info>,
        /// CHECK: Recipient, which should be the account owner of recipient token (read-only).
        ///
        /// NOTE: This used to be the rent sysvar. If the VAA encodes the recipient token account,
        /// this account does not need to be provided. Otherwise you need to provide this account,
        /// whose pubkey should match the VAA recipient.
        pub recipient: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct CompleteTransferWithPayloadNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Posted VAA Account (read-only, seeds = \["PostedVAA", message_hash\],
        /// seeds::program = core_bridge_program).
        pub vaa: AccountInfo<'info>,
        /// CHECK: Claim Account (mut, seeds = \[emitter_address, emitter_chain, sequence\],
        /// seeds::program = token_bridge_program).
        pub claim: AccountInfo<'info>,
        /// CHECK: Registered Emitter Account (mut, seeds = \[emitter_chain\], seeds::program =
        /// token_bridge_program).
        ///
        /// NOTE: If the above PDA does not exist, there is a legacy account whose address is
        /// derived using seeds = \[emitter_chain, emitter_address\].
        pub registered_emitter: AccountInfo<'info>,
        /// CHECK: Destination Token Account (mut).
        pub dst_token: AccountInfo<'info>,
        /// CHECK: Redeemer Authority (read-only signer).
        ///
        /// NOTE: In order to redeem a transfer sent to an address matching your program ID, use
        /// seeds = \["redeemer"\].
        pub redeemer_authority: AccountInfo<'info>,
        /// CHECK: Custody Token Account (mut, seeds = \[mint.key\], seeds::program =
        /// token_bridge_program).
        pub custody_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Custody Authority (read-only, seeds = \["custody_signer"\], seeds::program =
        /// token_bridge_program).
        pub custody_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct CompleteTransferWithPayloadWrapped<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Posted VAA Account (read-only, seeds = \["PostedVAA", message_hash\],
        /// seeds::program = core_bridge_program).
        pub vaa: AccountInfo<'info>,
        /// CHECK: Claim Account (mut, seeds = \[emitter_address, emitter_chain, sequence\],
        /// seeds::program = token_bridge_program).
        pub claim: AccountInfo<'info>,
        /// CHECK: Registered Emitter Account (mut, seeds = \[emitter_chain\], seeds::program =
        /// token_bridge_program).
        ///
        /// NOTE: If the above PDA does not exist, there is a legacy account whose address is
        /// derived using seeds = \[emitter_chain, emitter_address\].
        pub registered_emitter: AccountInfo<'info>,
        /// CHECK: Destination Token Account (mut).
        pub dst_token: AccountInfo<'info>,
        /// CHECK: Redeemer Authority (read-only signer).
        ///
        /// NOTE: In order to redeem a transfer sent to an address matching your program ID, use
        /// seeds = ["redeemer"].
        pub redeemer_authority: AccountInfo<'info>,
        /// CHECK: Wrapped Mint (mut, seeds = \["wrapped", token_chain, token_address\],
        /// seeds::program = token_bridge_program).
        pub wrapped_mint: AccountInfo<'info>,
        /// CHECK: Wrapped Asset (read-only, seeds = \["meta", wrapped_mint.key\], seeds::program =
        /// token_bridge_program).
        pub wrapped_asset: AccountInfo<'info>,
        /// CHECK: Mint Authority (read-only, seeds = \["mint_signer"\], seeds::program =
        /// token_bridge_program).
        pub mint_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
    }
    #[derive(Accounts)]
    pub struct TransferTokensNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Custody Token Account (mut, seeds = \[mint.key\], seeds::program =
        /// token_bridge_program).
        pub custody_token: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = \["authority_signer"\], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Custody Authority (read-only, seeds = \["custody_signer"\], seeds::program =
        /// token_bridge_program).
        pub custody_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = \["Bridge"\], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = \["emitter"\], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = \["fee_collector"\], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: SPL Token Program.
        pub token_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct TransferTokensWrapped<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Wrapped Mint (mut, seeds = \["wrapped", token_chain, token_address\],
        /// seeds::program = token_bridge_program).
        pub wrapped_mint: AccountInfo<'info>,
        /// CHECK: Wrapped Asset (read-only, seeds = ["meta", wrapped_mint.key], seeds::program =
        /// token_bridge_program).
        pub wrapped_asset: AccountInfo<'info>,
        /// CHECK: Transfer Authority (read-only, seeds = \["authority_signer"\], seeds::program =
        /// token_bridge_program).
        pub transfer_authority: AccountInfo<'info>,
        /// CHECK: Core Bridge Program Data (mut, seeds = \["Bridge"\], seeds::program =
        /// core_bridge_program).
        pub core_bridge_config: AccountInfo<'info>,
        /// CHECK: Core Bridge Message (mut).
        pub core_message: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter (read-only, seeds = \["emitter"\], seeds::program =
        /// token_bridge_program).
        pub core_emitter: AccountInfo<'info>,
        /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\],
        /// seeds::program = core_bridge_program).
        pub core_emitter_sequence: AccountInfo<'info>,
        /// CHECK: Core Bridge Fee Collector (mut, seeds = \["fee_collector"\], seeds::program =
        /// core_bridge_program).
        pub core_fee_collector: Option<AccountInfo<'info>>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: SPL Token Program.
        pub token_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
    }

    #[derive(Accounts)]
    pub struct TransferTokensWithPayloadNative<'info> {
        /// CHECK: Transaction payer (mut signer).
        pub payer: AccountInfo<'info>,
        /// CHECK: Source Token Account (mut).
        pub src_token: AccountInfo<'info>,
        /// CHECK: Mint (read-only).
        pub mint: AccountInfo<'info>,
        /// CHECK: Custody Token Account (mut, seeds = \[mint.key\], seeds::program =
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
        /// CHECK: Sender Authority (read-only signer).
        ///
        /// NOTE: In order for the program ID to be encoded as the sender address, use seeds =
        /// ["sender"] and specify cpi_program_id = Some(your_program_id).
        pub sender_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
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
        /// CHECK: Wrapped Asset (read-only, seeds = ["meta", wrapped_mint.key], seeds::program =
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
        /// CHECK: Sender Authority (read-only signer).
        ///
        /// NOTE: In order for the program ID to be encoded as the sender address, use seeds =
        /// ["sender"] and specify cpi_program_id = Some(your_program_id).
        pub sender_authority: AccountInfo<'info>,
        /// CHECK: System Program.
        pub system_program: AccountInfo<'info>,
        /// CHECK: Token Program.
        pub token_program: AccountInfo<'info>,
        /// CHECK: Core Bridge Program.
        pub core_bridge_program: AccountInfo<'info>,
    }
}
