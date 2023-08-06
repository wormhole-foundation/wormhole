//! Module containing the program’s set of instructions, where each method handler is associated
//! with a struct defining the input arguments to the method. These should be used directly, when
//! one wants to serialize instruction data, for example, when speciying instructions on a client.

mod initialize;
pub use initialize::InitializeArgs;

mod post_message;
#[cfg(feature = "no-entrypoint")]
pub use post_message::post_message;
pub use post_message::PostMessageArgs;

mod post_message_unreliable;
#[cfg(feature = "no-entrypoint")]
pub use post_message_unreliable::post_message_unreliable;

mod post_vaa;
pub use post_vaa::PostVaaArgs;

mod verify_signatures;
pub use verify_signatures::VerifySignaturesArgs;

use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// Legacy instruction selector.
///
/// NOTE: No more instructions should be added to this enum. Instead, add them as Anchor instruction
/// handlers, which will inevitably live in
/// [wormhole_core_bridge_solana](crate::wormhole_core_bridge_solana).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum LegacyInstruction {
    /// Initialize the program.
    Initialize,
    /// Publish a Wormhole message by creating a message account reflecting the message.
    PostMessage,
    /// Write an account reflecting a validated VAA (Version 1).
    PostVaa,
    /// **Governance.** Set the fee for posting a message.
    SetMessageFee,
    /// **Governance.** Collect Wormhole fees from the program's fee collector.
    TransferFees,
    /// **Governance.** Upgrade the program to a new implementation.
    UpgradeContract,
    /// **Governance.** Update the guardian set.
    GuardianSetUpdate,
    /// Verify guardian signatures of a VAA (Version 1).
    VerifySignatures,
    /// Publish a Wormhole message by either creating or reusing an existing message account.
    PostMessageUnreliable,
}

/// Unit struct used to represent an empty instruction argument.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct EmptyArgs {}
