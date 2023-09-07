use crate::{
    error::TokenBridgeError,
    legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs},
};
use anchor_lang::prelude::*;
use core_bridge_program::sdk::{self as core_bridge_sdk, LoadZeroCopy};

/// Trait for invoking one of the ways you can transfer assets to another network using the Token
/// Bridge program.
///
/// NOTE: A sender's address can either be a program's ID or a custom PDA address for transfers with
/// a message payload. If the sender address is a program ID, then the seeds for the sender
/// authority must be \["sender"\].
pub trait TransferTokens<'info>: core_bridge_sdk::cpi::PublishMessage<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info>;

    /// SPL Token Program.
    fn token_program(&self) -> AccountInfo<'info>;

    /// Core Bridge message account, which the Token Bridge program needs to publish its messages
    /// via Wormhole.
    fn core_message(&self) -> AccountInfo<'info>;

    /// Source Token Account (where tokens will be transferred from).
    fn src_token_account(&self) -> AccountInfo<'info>;

    /// Either of these types of mint:
    /// - Native (read-only).
    /// - Wrapped (mut, seeds = \["wrapped", token_chain, token_address\]).
    ///
    /// NOTE: If your instruction accepts either wrapped or native mints, you must specify this
    /// account with `#[account(mut)]`.
    fn mint(&self) -> AccountInfo<'info>;

    /// Transfer Authority (read-only, seeds = \["authority_signer"\]).
    fn token_bridge_transfer_authority(&self) -> AccountInfo<'info>;

    /// Custody Token Account (mut, seeds = \[mint.key\].
    ///
    /// NOTE: This must be specified as `Some(custody_token_account)` if the mint is native.
    fn token_bridge_custody_token_account(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Custody Authority (read-only, seeds = \["custody_signer"\]).
    ///
    /// NOTE: This must be specified as `Some(custody_authority)` if the mint is native.
    fn token_bridge_custody_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Wrapped Asset (read-only, seeds = \["meta", mint.key\].
    ///
    /// NOTE: This must be specified as `Some(wrapped_asset)` if the mint is wrapped.
    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Sender Authority (read-only signer). In order for the program ID to be encoded as the sender
    /// address, use seeds = ["sender"].
    ///
    /// NOTE: This must be specified as `Some(sender_authority)` if are using either
    /// [SignerTransferWithPayload](TransferTokensDirective::SignerTransferWithPayload) or
    /// [ProgramTransferWithPayload](TransferTokensDirective::ProgramTransferWithPayload).
    fn sender_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }
}

/// Direcrtive used to determine how to transfer assets.
pub enum TransferTokensDirective {
    /// Ordinary transfer with relay. If a relayer fee greater than zero is specified, this amount
    /// is deducted from the transfer amount to pay the redeemer of this transfer. This is useful to
    /// incentivize relayers to redeem your transfer (only if it is cost-effective for them to do
    /// so).
    Transfer {
        nonce: u32,
        amount: u64,
        relayer_fee: u64,
        recipient: [u8; 32],
        recipient_chain: u16,
    },
    /// Transfer with custom message payload. The sender address is the program ID of your program.
    ///
    /// NOTE: [sender_authority](TransferTokens::sender_authority) must use seeds = \["sender"\].
    ProgramTransferWithPayload {
        program_id: Pubkey,
        nonce: u32,
        amount: u64,
        redeemer: [u8; 32],
        redeemer_chain: u16,
        payload: Vec<u8>,
    },
    /// Transfer with custom message payload. The sender address is the pubkey of the
    /// [sender_authority](TransferTokens::sender_authority).
    SignerTransferWithPayload {
        nonce: u32,
        amount: u64,
        redeemer: [u8; 32],
        redeemer_chain: u16,
        payload: Vec<u8>,
    },
}

/// SDK method for transferring assets to another network with the Token Bridge program.
///
/// This method will handle any of the following directives:
///
/// * Transfer with relay.
/// * Transfer with message payload, whose sender is your program ID.
/// * Transfer with message payload, whose sender is either a keypair pubkey or PDA address.
///
/// Same requirements as [transfer_tokens_specified] except that this method will determine whether
/// the asset is wrapped or not by checking the [mint](TransferTokens::mint) account. **This will
/// cost a modest amount of compute units for the convenience of determining whether the mint is
/// Token Bridge wrapped or not.**
pub fn transfer_tokens<'info, A>(
    accounts: &A,
    directive: TransferTokensDirective,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    // If whether this mint is wrapped is unspecified, we derive the mint authority, which will cost
    // some compute units.
    let is_wrapped_asset =
        crate::utils::is_wrapped_mint(&crate::zero_copy::Mint::load(&accounts.mint())?);

    transfer_tokens_specified(accounts, directive, is_wrapped_asset, signer_seeds)
}

/// SDK method for transferring assets to another network with the Token Bridge program.
///
/// This method will handle any of the following directives:
/// * Transfer with relay.
/// * Transfer with message payload, whose sender is your program ID.
/// * Transfer with message payload, whose sender is either a keypair pubkey or PDA address.
///
/// The accounts must implement [TransferTokens].
///
/// Sender authority seeds are needed to act as a signer for the transfer with payload directives.
/// These seeds are either the seeds of a PDA or specifically seeds = \["sender"\] if the program ID
/// is the sender address.
///
/// Core message seeds are optional and are only needed if the integrating program is using a PDA
/// for this account. Otherwise, a keypair can be used.
pub fn transfer_tokens_specified<'info, A>(
    accounts: &A,
    directive: TransferTokensDirective,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    match directive {
        TransferTokensDirective::Transfer {
            nonce,
            amount,
            relayer_fee,
            recipient,
            recipient_chain,
        } => handle_transfer_tokens(
            accounts,
            TransferTokensArgs {
                nonce,
                amount,
                relayer_fee,
                recipient,
                recipient_chain,
            },
            is_wrapped_asset,
            signer_seeds,
        ),
        TransferTokensDirective::ProgramTransferWithPayload {
            program_id,
            nonce,
            amount,
            redeemer,
            redeemer_chain,
            payload,
        } => handle_transfer_tokens_with_payload(
            accounts,
            TransferTokensWithPayloadArgs {
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
                cpi_program_id: Some(program_id),
            },
            is_wrapped_asset,
            signer_seeds,
        ),
        TransferTokensDirective::SignerTransferWithPayload {
            nonce,
            amount,
            redeemer,
            redeemer_chain,
            payload,
        } => handle_transfer_tokens_with_payload(
            accounts,
            TransferTokensWithPayloadArgs {
                nonce,
                amount,
                redeemer,
                redeemer_chain,
                payload,
                cpi_program_id: None,
            },
            is_wrapped_asset,
            signer_seeds,
        ),
    }
}

fn handle_transfer_tokens<'info, A>(
    accounts: &A,
    args: TransferTokensArgs,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    if is_wrapped_asset {
        crate::legacy::cpi::transfer_tokens_wrapped(
            CpiContext::new_with_signer(
                accounts.token_bridge_program(),
                crate::legacy::cpi::TransferTokensWrapped {
                    payer: accounts.payer(),
                    src_token: accounts.src_token_account(),
                    wrapped_mint: accounts.mint(),
                    wrapped_asset: accounts
                        .token_bridge_wrapped_asset()
                        .ok_or(error!(TokenBridgeError::WrappedAssetRequired))?,
                    transfer_authority: accounts.token_bridge_transfer_authority(),
                    core_bridge_config: accounts.core_bridge_config(),
                    core_message: accounts.core_message(),
                    core_emitter: accounts.core_emitter_authority(),
                    core_emitter_sequence: accounts.core_emitter_sequence(),
                    core_fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                    token_program: accounts.token_program(),
                    core_bridge_program: accounts.core_bridge_program(),
                },
                signer_seeds.unwrap_or_default(),
            ),
            args,
        )
    } else {
        crate::legacy::cpi::transfer_tokens_native(
            CpiContext::new_with_signer(
                accounts.token_bridge_program(),
                crate::legacy::cpi::TransferTokensNative {
                    payer: accounts.payer(),
                    src_token: accounts.src_token_account(),
                    mint: accounts.mint(),
                    custody_token: accounts
                        .token_bridge_custody_token_account()
                        .ok_or(error!(TokenBridgeError::CustodyTokenAccountRequired))?,
                    transfer_authority: accounts.token_bridge_transfer_authority(),
                    custody_authority: accounts
                        .token_bridge_custody_authority()
                        .ok_or(error!(TokenBridgeError::CustodyAuthorityRequired))?,
                    core_bridge_config: accounts.core_bridge_config(),
                    core_message: accounts.core_message(),
                    core_emitter: accounts.core_emitter_authority(),
                    core_emitter_sequence: accounts.core_emitter_sequence(),
                    core_fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                    token_program: accounts.token_program(),
                    core_bridge_program: accounts.core_bridge_program(),
                },
                signer_seeds.unwrap_or_default(),
            ),
            args,
        )
    }
}

fn handle_transfer_tokens_with_payload<'info, A>(
    accounts: &A,
    args: TransferTokensWithPayloadArgs,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    if is_wrapped_asset {
        crate::legacy::cpi::transfer_tokens_with_payload_wrapped(
            CpiContext::new_with_signer(
                accounts.token_bridge_program(),
                crate::legacy::cpi::TransferTokensWithPayloadWrapped {
                    payer: accounts.payer(),
                    src_token: accounts.src_token_account(),
                    wrapped_mint: accounts.mint(),
                    wrapped_asset: accounts
                        .token_bridge_wrapped_asset()
                        .ok_or(error!(TokenBridgeError::WrappedAssetRequired))?,
                    transfer_authority: accounts.token_bridge_transfer_authority(),
                    core_bridge_config: accounts.core_bridge_config(),
                    core_message: accounts.core_message(),
                    core_emitter: accounts.core_emitter_authority(),
                    core_emitter_sequence: accounts.core_emitter_sequence(),
                    core_fee_collector: accounts.core_fee_collector(),
                    sender_authority: accounts
                        .sender_authority()
                        .ok_or(error!(TokenBridgeError::SenderAuthorityRequired))?,
                    system_program: accounts.system_program(),
                    token_program: accounts.token_program(),
                    core_bridge_program: accounts.core_bridge_program(),
                },
                signer_seeds.unwrap_or_default(),
            ),
            args,
        )
    } else {
        crate::legacy::cpi::transfer_tokens_with_payload_native(
            CpiContext::new_with_signer(
                accounts.token_bridge_program(),
                crate::legacy::cpi::TransferTokensWithPayloadNative {
                    payer: accounts.payer(),
                    src_token: accounts.src_token_account(),
                    mint: accounts.mint(),
                    custody_token: accounts
                        .token_bridge_custody_token_account()
                        .ok_or(error!(TokenBridgeError::CustodyTokenAccountRequired))?,
                    transfer_authority: accounts.token_bridge_transfer_authority(),
                    custody_authority: accounts
                        .token_bridge_custody_authority()
                        .ok_or(error!(TokenBridgeError::CustodyAuthorityRequired))?,
                    core_bridge_config: accounts.core_bridge_config(),
                    core_message: accounts.core_message(),
                    core_emitter: accounts.core_emitter_authority(),
                    core_emitter_sequence: accounts.core_emitter_sequence(),
                    core_fee_collector: accounts.core_fee_collector(),
                    sender_authority: accounts
                        .sender_authority()
                        .ok_or(error!(TokenBridgeError::SenderAuthorityRequired))?,
                    system_program: accounts.system_program(),
                    token_program: accounts.token_program(),
                    core_bridge_program: accounts.core_bridge_program(),
                },
                signer_seeds.unwrap_or_default(),
            ),
            args,
        )
    }
}
