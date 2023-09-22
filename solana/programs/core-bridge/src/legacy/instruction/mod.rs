//! Module containing the program’s set of instructions, where each method handler is associated
//! with a struct defining the input arguments to the method. These should be used directly, when
//! one wants to serialize instruction data, for example, when speciying instructions on a client.

use crate::types::Commitment;
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
    /// Write an account reflecting a verified VAA (Version 1).
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

/// Arguments used to initialize the Core Bridge program.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitializeArgs {
    pub guardian_set_ttl_seconds: u32,
    pub fee_lamports: u64,
    pub initial_guardians: Vec<[u8; 20]>,
}

/// Arguments used to post a new Wormhole (Core Bridge) message either using
/// [post_message](crate::legacy::instruction::post_message) or
/// [post_message_unreliable](crate::legacy::instruction::post_message_unreliable).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct PostMessageArgs {
    /// Unique id for this message.
    pub nonce: u32,
    /// Encoded message.
    pub payload: Vec<u8>,
    /// Solana commitment level for Guardian observation.
    pub commitment: Commitment,
}

/// Arguments to post new VAA data after signature verification.
///
/// NOTE: It is preferred to use the new process of verifying a VAA using the new Core Bridge Anchor
/// instructions. See [init_encoded_vaa](crate::wormhole_core_bridge_solana::init_encoded_vaa) and
/// [write_encoded_vaa](crate::wormhole_core_bridge_solana::write_encoded_vaa) for more info.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct PostVaaArgs {
    /// Unused data.
    pub _gap_0: [u8; 5],
    /// Time the message was submitted.
    pub timestamp: u32,
    /// Unique ID for this message.
    pub nonce: u32,
    /// The Wormhole chain ID denoting the origin of this message.
    pub emitter_chain: u16,
    /// Emitter of the message.
    pub emitter_address: [u8; 32],
    /// Sequence number of this message.
    pub sequence: u64,
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,
    /// Message payload.
    pub payload: Vec<u8>,
}

/// Arguments to verify specific guardian indices.
///
/// NOTE: It is preferred to use the new process of verifying a VAA using the new Core Bridge Anchor
/// instructions. See [init_encoded_vaa](crate::wormhole_core_bridge_solana::init_encoded_vaa) and
/// [write_encoded_vaa](crate::wormhole_core_bridge_solana::write_encoded_vaa) for more info.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct VerifySignaturesArgs {
    /// Indices of verified guardian signatures, where -1 indicates a missing value. There is a
    /// missing value if the guardian at this index is not expected to have its signature verfied by
    /// the Sig Verify native program in the instruction invoked prior).
    ///
    /// NOTE: In the legacy implementation, this argument being a fixed-sized array of 19 only
    /// allows the first 19 guardians of any size guardian set to be verified. Because of this, it
    /// is absolutely important to use the new process of verifying a VAA.
    pub signer_indices: [i8; 19],
}

/// Unit struct used to represent an empty instruction argument.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct EmptyArgs {}

#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::{LegacyInstruction, PostMessageArgs};
    use anchor_lang::ToAccountMetas;
    use solana_program::instruction::Instruction;

    /// Processor to post (publish) a Wormhole message by setting up the message account for
    /// Guardian observation.
    ///
    /// A message is either created beforehand using the new Anchor instruction to process a message
    /// or is created at this point.
    pub fn post_message(
        accounts: crate::legacy::accounts::PostMessage,
        args: PostMessageArgs,
    ) -> Instruction {
        let message_is_signer = !args.payload.is_empty();
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::PostMessage, args),
            accounts.to_account_metas(Some(message_is_signer)),
        )
    }

    /// Processor to post (publish) a Wormhole message by setting up the message account for
    /// Guardian observation. This message account has either been created already or is created in
    /// this call.
    ///
    /// If this message account already exists, the emitter must be the same as the one encoded in
    /// the message and the payload must be the same size.
    pub fn post_message_unreliable(
        accounts: crate::legacy::accounts::PostMessageUnreliable,
        args: PostMessageArgs,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::PostMessageUnreliable, args),
            accounts.to_account_metas(None),
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
