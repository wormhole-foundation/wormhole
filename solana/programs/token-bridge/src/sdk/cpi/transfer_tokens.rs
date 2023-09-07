pub use crate::legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs};

use anchor_lang::prelude::*;

pub trait TransferTokens<'info>:
    super::InvokeTokenBridge<'info> + core_bridge_program::sdk::cpi::PublishMessage<'info>
{
    fn src_token_account(&self) -> AccountInfo<'info>;

    fn mint(&self) -> AccountInfo<'info>;

    fn transfer_authority(&self) -> AccountInfo<'info>;

    fn custody_token_account(&self) -> Option<AccountInfo<'info>>;

    fn custody_authority(&self) -> Option<AccountInfo<'info>>;

    fn wrapped_asset(&self) -> Option<AccountInfo<'info>>;

    fn sender_authority(&self) -> Option<AccountInfo<'info>>;
}

pub enum TransferTokensDirective {
    Transfer {
        nonce: u32,
        amount: u64,
        relayer_fee: u64,
        recipient: [u8; 32],
        recipient_chain: u16,
    },
    SignerTransferWithPayload {
        nonce: u32,
        amount: u64,
        redeemer: [u8; 32],
        redeemer_chain: u16,
        payload: Vec<u8>,
    },
    ProgramTransferWithPayload {
        program_id: Pubkey,
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
            signer_seeds,
        ),
    }
}

pub fn handle_transfer_tokens<'info, A>(
    _accounts: &A,
    _args: TransferTokensArgs,
    _signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    // TODO
    Ok(())
}

pub fn handle_transfer_tokens_with_payload<'info, A>(
    _accounts: &A,
    _args: TransferTokensWithPayloadArgs,
    _signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: TransferTokens<'info>,
{
    // TODO
    Ok(())
}
