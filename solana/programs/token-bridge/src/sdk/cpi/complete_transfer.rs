use crate::{error::TokenBridgeError, zero_copy::Mint};
use anchor_lang::prelude::*;
use core_bridge_program::sdk::LoadZeroCopy;

pub trait CompleteTransfer<'info>: super::system_program::CreateAccount<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info>;

    /// SPL Token Program.
    fn token_program(&self) -> AccountInfo<'info>;

    fn vaa(&self) -> AccountInfo<'info>;

    fn token_bridge_claim(&self) -> AccountInfo<'info>;

    fn token_bridge_registered_emitter(&self) -> AccountInfo<'info>;

    /// Destination Token Account (where tokens will be transferred to). For regular transfers, this
    /// account belongs to the recipient, which may be encoded in the VAA as the recipient.
    fn dst_token_account(&self) -> AccountInfo<'info>;

    /// Either of these types of mint:
    /// - Native (read-only).
    /// - Wrapped (mut, seeds = \["wrapped", token_chain, token_address\]).
    fn mint(&self) -> AccountInfo<'info>;

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

    fn token_bridge_mint_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Redeemer Authority (read-only signer). In order to redeem a transfer as your program (the
    /// encoded redeemer address is your program ID), use seeds = ["redeemer"]. Otherwise the
    /// redeemer address of your custom signer must be specified (which will be the pubkey of this
    /// account).
    ///
    /// NOTE: This account must be specified as `Some(redeemer_authority)` if the transfer redeemed
    /// has a message payload associated with it.
    fn redeemer_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Relayer's token account.
    fn payer_token(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Recipient, who is the owner of [dst_token_account](CompleteTransfer::dst_token_account).
    /// This account only matters if the transfer redeemed is a relayable transfer (i.e. no message
    /// associated with the transfer).
    fn recipient(&self) -> Option<AccountInfo<'info>> {
        None
    }
}

pub fn complete_transfer<'info, A>(accounts: &A, signer_seeds: Option<&[&[&[u8]]]>) -> Result<()>
where
    A: CompleteTransfer<'info>,
{
    // If whether this mint is wrapped is unspecified, we derive the mint authority, which will cost
    // some compute units.
    let is_wrapped_asset = crate::utils::is_wrapped_mint(&Mint::load(&accounts.mint())?);

    complete_transfer_specified(accounts, is_wrapped_asset, signer_seeds)
}

pub fn complete_transfer_specified<'info, A>(
    accounts: &A,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CompleteTransfer<'info>,
{
    match accounts.redeemer_authority() {
        Some(_) => handle_complete_transfer_with_payload(accounts, is_wrapped_asset, signer_seeds),
        None => handle_complete_transfer(accounts, is_wrapped_asset, signer_seeds),
    }
}

fn handle_complete_transfer<'info, A>(
    accounts: &A,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CompleteTransfer<'info>,
{
    let payer_token = accounts
        .payer_token()
        .ok_or(error!(TokenBridgeError::PayerTokenRequired))?;

    if is_wrapped_asset {
        crate::legacy::cpi::complete_transfer_wrapped(CpiContext::new_with_signer(
            accounts.token_bridge_program(),
            crate::legacy::cpi::CompleteTransferWrapped {
                payer: accounts.payer(),
                vaa: accounts.vaa(),
                claim: accounts.token_bridge_claim(),
                registered_emitter: accounts.token_bridge_registered_emitter(),
                recipient_token: accounts.dst_token_account(),
                payer_token,
                wrapped_mint: accounts.mint(),
                wrapped_asset: accounts
                    .token_bridge_wrapped_asset()
                    .ok_or(error!(TokenBridgeError::WrappedAssetRequired))?,
                mint_authority: accounts
                    .token_bridge_mint_authority()
                    .ok_or(error!(TokenBridgeError::MintAuthorityRequired))?,
                recipient: accounts.recipient(),
                system_program: accounts.system_program(),
                token_program: accounts.token_program(),
            },
            signer_seeds.unwrap_or_default(),
        ))
    } else {
        crate::legacy::cpi::complete_transfer_native(CpiContext::new_with_signer(
            accounts.token_bridge_program(),
            crate::legacy::cpi::CompleteTransferNative {
                payer: accounts.payer(),
                vaa: accounts.vaa(),
                claim: accounts.token_bridge_claim(),
                registered_emitter: accounts.token_bridge_registered_emitter(),
                recipient_token: accounts.dst_token_account(),
                payer_token,
                custody_token: accounts
                    .token_bridge_custody_token_account()
                    .ok_or(error!(TokenBridgeError::CustodyTokenAccountRequired))?,
                mint: accounts.mint(),
                custody_authority: accounts
                    .token_bridge_custody_authority()
                    .ok_or(error!(TokenBridgeError::CustodyAuthorityRequired))?,
                recipient: accounts.recipient(),
                system_program: accounts.system_program(),
                token_program: accounts.token_program(),
            },
            signer_seeds.unwrap_or_default(),
        ))
    }
}

fn handle_complete_transfer_with_payload<'info, A>(
    accounts: &A,
    is_wrapped_asset: bool,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CompleteTransfer<'info>,
{
    let redeemer_authority = accounts
        .redeemer_authority()
        .ok_or(error!(TokenBridgeError::RedeemerAuthorityRequired))?;

    if is_wrapped_asset {
        crate::legacy::cpi::complete_transfer_with_payload_wrapped(CpiContext::new_with_signer(
            accounts.token_bridge_program(),
            crate::legacy::cpi::CompleteTransferWithPayloadWrapped {
                payer: accounts.payer(),
                vaa: accounts.vaa(),
                claim: accounts.token_bridge_claim(),
                registered_emitter: accounts.token_bridge_registered_emitter(),
                dst_token: accounts.dst_token_account(),
                redeemer_authority,
                wrapped_mint: accounts.mint(),
                wrapped_asset: accounts
                    .token_bridge_wrapped_asset()
                    .ok_or(error!(TokenBridgeError::WrappedAssetRequired))?,
                mint_authority: accounts
                    .token_bridge_mint_authority()
                    .ok_or(error!(TokenBridgeError::MintAuthorityRequired))?,
                system_program: accounts.system_program(),
                token_program: accounts.token_program(),
            },
            signer_seeds.unwrap_or_default(),
        ))
    } else {
        crate::legacy::cpi::complete_transfer_with_payload_native(CpiContext::new_with_signer(
            accounts.token_bridge_program(),
            crate::legacy::cpi::CompleteTransferWithPayloadNative {
                payer: accounts.payer(),
                vaa: accounts.vaa(),
                claim: accounts.token_bridge_claim(),
                registered_emitter: accounts.token_bridge_registered_emitter(),
                dst_token: accounts.dst_token_account(),
                redeemer_authority,
                custody_token: accounts
                    .token_bridge_custody_token_account()
                    .ok_or(error!(TokenBridgeError::CustodyTokenAccountRequired))?,
                mint: accounts.mint(),
                custody_authority: accounts
                    .token_bridge_custody_authority()
                    .ok_or(error!(TokenBridgeError::CustodyAuthorityRequired))?,
                system_program: accounts.system_program(),
                token_program: accounts.token_program(),
            },
            signer_seeds.unwrap_or_default(),
        ))
    }
}
