mod initialize;
mod post_message;
mod post_message_unreliable;
mod post_vaa;
mod verify_signatures;

pub use initialize::*;
pub use post_message::*;
pub use post_message_unreliable::*;
pub use post_vaa::*;
pub use verify_signatures::*;

use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// NOTE: No more instructions should be added to this enum. Instead, add them as Anchor instruction
/// handlers, which will inevitably live in lib.rs.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum LegacyInstruction {
    Initialize,
    PostMessage,
    PostVaa,
    SetMessageFee,
    TransferFees,
    UpgradeContract,
    GuardianSetUpdate,
    VerifySignatures,
    PostMessageUnreliable,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct EmptyArgs {}
