use crate::{legacy::instruction::PostMessageArgs, types::Commitment};
use anchor_lang::prelude::*;

/// Trait for invoking one of the ways you can publish a Wormhole message using the Core Bridge
/// program.
///
/// NOTE: A message's emitter address can either be a program's ID or a custom address (determined
/// by either a keypair or program's PDA). If the emitter address is a program ID, then
/// the seeds for the emitter authority must be \["emitter"\].
pub trait PublishMessage<'info>: super::system_program::CreateAccount<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info>;

    /// Core Bridge Emitter Authority (read-only signer). This account should return either the
    /// account that will act as the emitter address or the signer for a program emitter, where the
    /// emitter address is a program ID. This emitter also acts as the authority for preparing a
    /// message before it is posted.
    fn core_emitter_authority(&self) -> AccountInfo<'info>;

    /// Core Bridge Program Data (mut, seeds = \["Bridge"\]).
    fn core_bridge_config(&self) -> AccountInfo<'info>;

    /// Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
    fn core_emitter_sequence(&self) -> AccountInfo<'info>;

    /// Core Bridge Fee Collector (mut, seeds = \["fee_collector"\]).
    ///
    /// NOTE: This account is mutable because the SDK method that publishes messages pays the
    /// Wormhole fee, which requires a lamport transfer from the payer to the fee collector.
    fn core_fee_collector(&self) -> Option<AccountInfo<'info>>;
}

/// Directive used to determine how to post a Core Bridge message.
pub enum PublishMessageDirective {
    /// Ordinary message, which creates a new account for the Core Bridge message. The emitter
    /// address is the pubkey of the emitter signer.
    Message {
        nonce: u32,
        payload: Vec<u8>,
        commitment: Commitment,
    },
    /// Ordinary message, which creates a new account for the Core Bridge message. The emitter
    /// address is the program ID specified in this directive.
    ///
    /// NOTE: [core_emitter_authority](PublishMessage::core_emitter_authority) must use seeds =
    /// \["emitter"\].
    ProgramMessage {
        program_id: Pubkey,
        nonce: u32,
        payload: Vec<u8>,
        commitment: Commitment,
    },
    /// Unreliable (reusable) message, which will either create a new account or reuse an existing
    /// Core Bridge message account. The emitter address is the pubkey of the emitter signer. If a
    /// message account is reused, the payload length must be the same as the existing message's.
    UnreliableMessage {
        nonce: u32,
        payload: Vec<u8>,
        commitment: Commitment,
    },
    /// Prepared message, which was already written to a message account. Usually this operation
    /// would be executed within a transaction block following a program's instruction(s) preparing
    /// a Wormhole message. But if there are other operations your program needs to perform in
    /// concert with publishing a prepared message, use this directive.
    PreparedMessage,
}

/// SDK method for posting a new message with the Core Bridge program.
///
/// This method will handle any of the following directives:
///
/// * Post a new message with an emitter address determined by either a keypair pubkey or PDA
///   address.
/// * Post a new message with an emitter address that is a program ID.
/// * Post an unreliable message, which can reuse a message account with a new payload.
///
/// The accounts must implement [PublishMessage].
///
/// Emitter seeds are needed to act as a signer for the post message instructions. These seeds are
/// either the seeds of a PDA or specifically seeds = \["emitter"\] if the program ID is the
/// emitter address.
///
/// Message seeds are optional and are only needed if the integrating program is using a PDA for
/// this account. Otherwise, a keypair can be used.
pub fn publish_message<'info, A>(
    accounts: &A,
    new_message: &AccountInfo<'info>,
    directive: PublishMessageDirective,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    // If there is a fee, transfer it. But only try if the fee collector is provided because the
    // post message instruction will fail if there is actually a fee but no fee collector.
    if let Some(fee_collector) = accounts.core_fee_collector() {
        let fee_lamports = crate::zero_copy::Config::parse(&accounts.core_bridge_config())
            .map(|config| config.fee_lamports())?;

        if fee_lamports > 0 {
            anchor_lang::system_program::transfer(
                CpiContext::new(
                    accounts.system_program(),
                    anchor_lang::system_program::Transfer {
                        from: accounts.payer(),
                        to: fee_collector,
                    },
                ),
                fee_lamports,
            )?;
        }
    }

    match directive {
        PublishMessageDirective::Message {
            nonce,
            payload,
            commitment,
        } => handle_post_message_v1(
            accounts,
            new_message,
            PostMessageArgs {
                nonce,
                payload,
                commitment,
            },
            signer_seeds,
        ),
        PublishMessageDirective::ProgramMessage {
            program_id,
            nonce,
            payload,
            commitment,
        } => handle_post_program_message_v1(
            accounts,
            new_message,
            program_id,
            nonce,
            payload,
            commitment,
            signer_seeds,
        ),
        PublishMessageDirective::UnreliableMessage {
            nonce,
            payload,
            commitment,
        } => handle_post_unreliable_message_v1(
            accounts,
            new_message,
            PostMessageArgs {
                nonce,
                payload,
                commitment,
            },
            signer_seeds,
        ),
        PublishMessageDirective::PreparedMessage => {
            handle_prepared_message_v1(accounts, new_message, signer_seeds)
        }
    }
}

fn handle_post_message_v1<'info, A>(
    accounts: &A,
    new_message: &AccountInfo<'info>,
    args: PostMessageArgs,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    crate::legacy::cpi::post_message(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::legacy::cpi::PostMessage {
                config: accounts.core_bridge_config(),
                message: new_message.to_account_info(),
                emitter: Some(accounts.core_emitter_authority()),
                emitter_sequence: accounts.core_emitter_sequence(),
                payer: accounts.payer(),
                fee_collector: accounts.core_fee_collector(),
                system_program: accounts.system_program(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        args,
    )
}

fn handle_post_program_message_v1<'info, A>(
    accounts: &A,
    new_message: &AccountInfo<'info>,
    program_id: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: Commitment,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    // Create message account.
    crate::utils::cpi::create_account(
        accounts,
        new_message,
        crate::sdk::compute_prepared_message_space(payload.len()),
        &crate::ID,
        signer_seeds,
    )?;

    // Prepare (calling init and process instructions).
    crate::sdk::cpi::handle_prepare_message_v1(
        accounts.core_bridge_program(),
        new_message.to_account_info(),
        accounts.core_emitter_authority(),
        crate::sdk::cpi::InitMessageV1Args {
            nonce,
            cpi_program_id: Some(program_id),
            commitment,
        },
        payload,
        signer_seeds,
    )?;

    // Finally post.
    crate::legacy::cpi::post_message(
        CpiContext::new(
            accounts.core_bridge_program(),
            crate::legacy::cpi::PostMessage {
                config: accounts.core_bridge_config(),
                message: new_message.to_account_info(),
                emitter: None,
                emitter_sequence: accounts.core_emitter_sequence(),
                payer: accounts.payer(),
                fee_collector: accounts.core_fee_collector(),
                system_program: accounts.system_program(),
            },
        ),
        PostMessageArgs {
            nonce: Default::default(), // not checked
            payload: Vec::new(),
            commitment: crate::types::Commitment::Finalized, // not checked
        },
    )
}

fn handle_post_unreliable_message_v1<'info, A>(
    accounts: &A,
    new_message: &AccountInfo<'info>,
    args: PostMessageArgs,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    crate::legacy::cpi::post_message_unreliable(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::legacy::cpi::PostMessageUnreliable {
                config: accounts.core_bridge_config(),
                message: new_message.to_account_info(),
                emitter: accounts.core_emitter_authority(),
                emitter_sequence: accounts.core_emitter_sequence(),
                payer: accounts.payer(),
                fee_collector: accounts.core_fee_collector(),
                system_program: accounts.system_program(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        args,
    )
}

fn handle_prepared_message_v1<'info, A>(
    accounts: &A,
    new_message: &AccountInfo<'info>,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    crate::legacy::cpi::post_message(
        CpiContext::new_with_signer(
            accounts.core_bridge_program(),
            crate::legacy::cpi::PostMessage {
                config: accounts.core_bridge_config(),
                message: new_message.to_account_info(),
                emitter: None,
                emitter_sequence: accounts.core_emitter_sequence(),
                payer: accounts.payer(),
                fee_collector: accounts.core_fee_collector(),
                system_program: accounts.system_program(),
            },
            signer_seeds.unwrap_or_default(),
        ),
        PostMessageArgs {
            nonce: 420, // not checked
            payload: Vec::new(),
            commitment: Commitment::Finalized, // not checked
        },
    )
}
