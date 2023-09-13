pub use crate::legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs};

use crate::{error::TokenBridgeError, zero_copy};
use anchor_lang::prelude::*;

pub trait TransferTokens<'info>: core_bridge_program::sdk::cpi::PublishMessage<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info>;

    /// SPL Token Program.
    fn token_program(&self) -> AccountInfo<'info>;

    /// Source Token Account (where tokens will be transferred from).
    fn src_token_account(&self) -> AccountInfo<'info>;

    /// Either of these types of mint:
    /// - Native (read-only).
    /// - Wrapped (mut, seeds = \["wrapped", token_chain, token_address\]).
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
    fn token_bridge_sender_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Try unwrapping [custody_token_account](TransferTokens::token_bridge_custody_token_account).
    fn try_custody_token_account(&self) -> Result<AccountInfo<'info>> {
        self.token_bridge_custody_token_account()
            .ok_or(error!(TokenBridgeError::CustodyTokenAccountRequired))
    }

    /// Try unwrapping [custody_authority](TransferTokens::token_bridge_custody_authority).
    fn try_custody_authority(&self) -> Result<AccountInfo<'info>> {
        self.token_bridge_custody_authority()
            .ok_or(error!(TokenBridgeError::CustodyAuthorityRequired))
    }

    /// Try unwrapping [wrapped_asset](TransferTokens::token_bridge_wrapped_asset).
    fn try_wrapped_asset(&self) -> Result<AccountInfo<'info>> {
        self.token_bridge_wrapped_asset()
            .ok_or(error!(TokenBridgeError::WrappedAssetRequired))
    }

    /// Try unwrapping [sender_authority](TransferTokens::token_bridge_sender_authority).
    fn try_sender_authority(&self) -> Result<AccountInfo<'info>> {
        self.token_bridge_sender_authority()
            .ok_or(error!(TokenBridgeError::SenderAuthorityRequired))
    }
}

pub enum TransferTokensDirective {
    Transfer {
        nonce: u32,
        amount: u64,
        relayer_fee: u64,
        recipient: [u8; 32],
        recipient_chain: u16,
    },
    ProgramTransferWithPayload {
        program_id: Pubkey,
        nonce: u32,
        amount: u64,
        redeemer: [u8; 32],
        redeemer_chain: u16,
        payload: Vec<u8>,
    },
    SignerTransferWithPayload {
        nonce: u32,
        amount: u64,
        redeemer: [u8; 32],
        redeemer_chain: u16,
        payload: Vec<u8>,
    },
}

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
    let is_wrapped_asset = {
        let mint = accounts.mint();
        let acc_data = mint.try_borrow_data()?;
        let mint = zero_copy::Mint::parse(&acc_data)?;
        crate::utils::is_native_mint(&mint)
    };

    transfer_tokens_specified(accounts, directive, is_wrapped_asset, signer_seeds)
}

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
        match signer_seeds {
            Some(signer_seeds) => crate::legacy::cpi::transfer_tokens_wrapped(
                CpiContext::new_with_signer(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWrapped {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        wrapped_mint: accounts.mint(),
                        wrapped_asset: accounts.try_wrapped_asset()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                    signer_seeds,
                ),
                args,
            ),
            None => crate::legacy::cpi::transfer_tokens_wrapped(
                CpiContext::new(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWrapped {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        wrapped_mint: accounts.mint(),
                        wrapped_asset: accounts.try_wrapped_asset()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                ),
                args,
            ),
        }
    } else {
        match signer_seeds {
            Some(signer_seeds) => crate::legacy::cpi::transfer_tokens_native(
                CpiContext::new_with_signer(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensNative {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        mint: accounts.mint(),
                        custody_token: accounts.try_custody_token_account()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        custody_authority: accounts.try_custody_authority()?,
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                    signer_seeds,
                ),
                args,
            ),
            None => crate::legacy::cpi::transfer_tokens_native(
                CpiContext::new(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensNative {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        mint: accounts.mint(),
                        custody_token: accounts.try_custody_token_account()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        custody_authority: accounts.try_custody_authority()?,
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                ),
                args,
            ),
        }
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
        match signer_seeds {
            Some(signer_seeds) => crate::legacy::cpi::transfer_tokens_with_payload_wrapped(
                CpiContext::new_with_signer(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWithPayloadWrapped {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        wrapped_mint: accounts.mint(),
                        wrapped_asset: accounts.try_wrapped_asset()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        sender_authority: accounts.try_sender_authority()?,
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                    signer_seeds,
                ),
                args,
            ),
            None => crate::legacy::cpi::transfer_tokens_with_payload_wrapped(
                CpiContext::new(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWithPayloadWrapped {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        wrapped_mint: accounts.mint(),
                        wrapped_asset: accounts.try_wrapped_asset()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        sender_authority: accounts.try_sender_authority()?,
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                ),
                args,
            ),
        }
    } else {
        match signer_seeds {
            Some(signer_seeds) => crate::legacy::cpi::transfer_tokens_with_payload_native(
                CpiContext::new_with_signer(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWithPayloadNative {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        mint: accounts.mint(),
                        custody_token: accounts.try_custody_token_account()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        custody_authority: accounts.try_custody_authority()?,
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        sender_authority: accounts.try_sender_authority()?,
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                    signer_seeds,
                ),
                args,
            ),
            None => crate::legacy::cpi::transfer_tokens_with_payload_native(
                CpiContext::new(
                    accounts.token_bridge_program(),
                    crate::legacy::cpi::TransferTokensWithPayloadNative {
                        payer: accounts.payer(),
                        src_token: accounts.src_token_account(),
                        mint: accounts.mint(),
                        custody_token: accounts.try_custody_token_account()?,
                        transfer_authority: accounts.token_bridge_transfer_authority(),
                        custody_authority: accounts.try_custody_authority()?,
                        core_bridge_config: accounts.core_bridge_config(),
                        core_message: accounts.core_message(),
                        core_emitter: accounts.try_core_emitter()?,
                        core_emitter_sequence: accounts.core_emitter_sequence(),
                        core_fee_collector: accounts.core_fee_collector(),
                        sender_authority: accounts.try_sender_authority()?,
                        system_program: accounts.system_program(),
                        token_program: accounts.token_program(),
                        core_bridge_program: accounts.core_bridge_program(),
                    },
                ),
                args,
            ),
        }
    }
}
