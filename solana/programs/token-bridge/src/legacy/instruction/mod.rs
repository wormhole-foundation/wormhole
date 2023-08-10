mod attest_token;
mod initialize;
mod transfer_tokens;
mod transfer_tokens_with_payload;

pub use attest_token::*;
pub use initialize::*;
pub use transfer_tokens::*;
pub use transfer_tokens_with_payload::*;

use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// NOTE: No more instructions should be added to this enum. Instead, add them as Anchor instruction
/// handlers, which will inevitably live in lib.rs.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum LegacyInstruction {
    Initialize,
    AttestToken,
    CompleteTransferNative,
    CompleteTransferWrapped,
    TransferTokensWrapped,
    TransferTokensNative,
    RegisterChain,
    CreateOrUpdateWrapped,
    UpgradeContract,
    CompleteTransferWithPayloadNative,
    CompleteTransferWithPayloadWrapped,
    TransferTokensWithPayloadWrapped,
    TransferTokensWithPayloadNative,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct EmptyArgs {}
