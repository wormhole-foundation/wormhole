pub use crate::legacy::instruction::PostMessageArgs;

use crate::{error::CoreBridgeError, types::Commitment};
use anchor_lang::prelude::*;

/// Trait for invoking one of the ways you can publish a Wormhole message using the Core Bridge
/// program.
///
/// A message's emitter address can either based on a program's ID or a custom address (determined
/// by either a keypair or program's PDA). Depending on which emitter address is used, the
/// [core_emitter](PublishMessage::core_emitter) or
/// [core_emitter_authority](PublishMessage::core_emitter_authority) account must be provided.
///
/// When the emitter itself is a signer for the post message instruction, you must specify `Some`
/// for [core_emitter](PublishMessage::core_emitter). Otherwise, if the emitter is a program ID, you
/// must specify Some for [core_emitter_authority](PublishMessage::core_emitter_authority), which is
/// the program's authority to draft a new message to prepare it for posting. By default,
/// [core_emitter_authority](PublishMessage::core_emitter_authority) returns `None`, so you must
/// override it if the emitter address is a program ID.
pub trait PublishMessage<'info>:
    super::InvokeCoreBridge<'info> + super::CreateAccount<'info>
{
    /// Core Bridge Program Data (mut, seeds = \["Bridge"\]).
    fn core_bridge_config(&self) -> AccountInfo<'info>;

    /// Core Bridge Message (mut).
    fn core_message(&self) -> AccountInfo<'info>;

    /// Core Bridge Emitter (read-only signer).
    ///
    /// NOTE: This account isn't checked if the message's emitter address is a program ID, so
    /// in this case it can return None.
    fn core_emitter(&self) -> Option<AccountInfo<'info>>;

    /// Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
    fn core_emitter_sequence(&self) -> AccountInfo<'info>;

    /// Core Bridge Fee Collector (mut, seeds = \["fee_collector"\]).
    ///
    /// NOTE: This account is mutable because the SDK method that publishes messages pays the
    /// Wormhole fee, which requires a lamport transfer from the payer to the fee collector.
    fn core_fee_collector(&self) -> Option<AccountInfo<'info>>;

    /// Core Bridge Emitter Authority (read-only signer). This account should return Some if the
    /// emitter address is a program ID. This emitter authority acts as the signer for preparing a
    /// message before it is posted.
    fn core_emitter_authority(&self) -> Option<AccountInfo<'info>> {
        None
    }

    /// Try unwrapping [core_emitter](PublishMessage::core_emitter).
    fn try_core_emitter(&self) -> Result<AccountInfo<'info>> {
        self.core_emitter()
            .ok_or(error!(CoreBridgeError::EmitterRequired))
    }

    /// Try unwrapping [core_emitter_authority](PublishMessage::core_emitter_authority).
    fn try_core_emitter_authority(&self) -> Result<AccountInfo<'info>> {
        self.core_emitter_authority()
            .ok_or(error!(CoreBridgeError::EmitterAuthorityRequired))
    }
}

/// Directive used to determine how to post a Core Bridge message.
pub enum PublishMessageDirective {
    /// Ordinary message, which creates a new account for the Core Bridge message. The emitter
    /// address is the pubkey of the emitter signer.
    ///
    /// NOTE: The core_emitter in [PublishMessage] must return `Some`, which will be the account
    /// info for the emitter signer. See [post_message](crate::legacy::cpi::post_message) for more
    /// info.
    Message {
        nonce: u32,
        payload: Vec<u8>,
        commitment: Commitment,
    },
    /// Ordinary message, which creates a new account for the Core Bridge message. The emitter
    /// address is the program ID specified in this directive.
    ///
    /// NOTE: The core_emitter_authority in [PublishMessage] must return `Some`, which will be the
    /// account info for the authority used to prepare a new draft message. See
    /// [init_message_v1](crate::cpi::init_message_v1) and
    /// [process_message_v1](crate::cpi::process_message_v1) for more details.
    ProgramMessage {
        program_id: Pubkey,
        nonce: u32,
        payload: Vec<u8>,
        commitment: Commitment,
    },
    /// Unreliable (reusable) message, which will either create a new account or reuse an existing
    /// Core Bridge message account. The emitter address is the pubkey of the emitter signer. If a
    /// message account is reused, the payload length must be the same as the existing message's.
    ///
    /// NOTE: The core_emitter in [PublishMessage] must return `Some`, which will be the account
    /// info for the emitter signer. See [post_message](crate::legacy::cpi::post_message) for more
    /// info.
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

/// SDK method for posting a new message with the Core Bridge program. This method will handle any
/// of the following directives:
/// * Post a new message with an emitter address determined by either a keypair or program PDA.
/// * Post a new message with an emitter address that is a program ID.
/// * Post an unreliable message, which can reuse a message account with a new payload.
///
/// The accounts must implement [PublishMessage].
///
/// Emitter seeds are needed to act as a signer for the post message instructions. These seeds are
/// either the seeds of a program's PDA or specifically seeds = \["emitter"\] if the program ID is the
/// emitter address.
///
/// Message seeds are optional and are only needed if the integrating program is using a PDA for
/// this account. Otherwise, a keypair can be used and message seeds can be None.
pub fn publish_message<'info, A>(
    accounts: &A,
    directive: PublishMessageDirective,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    // If there is a fee, transfer it. But only try if the fee collector is provided because the
    // post message instruction will fail if there is actually a fee but no fee collector.
    if let Some(fee_collector) = accounts.core_fee_collector() {
        let fee_lamports =
            crate::zero_copy::Config::parse(&accounts.core_bridge_config().try_borrow_data()?)?
                .fee_lamports();

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
            PostMessageArgs {
                nonce,
                payload,
                commitment,
            },
            signer_seeds,
        ),
        PublishMessageDirective::PreparedMessage => {
            handle_prepared_message_v1(accounts, signer_seeds)
        }
    }
}

fn handle_post_message_v1<'info, A>(
    accounts: &A,
    args: PostMessageArgs,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    match signer_seeds {
        Some(signer_seeds) => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: accounts.core_emitter(),
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
                signer_seeds,
            ),
            args,
        ),
        None => crate::legacy::cpi::post_message(
            CpiContext::new(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: accounts.core_emitter(),
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
            ),
            args,
        ),
    }
}

fn handle_post_program_message_v1<'info, A>(
    accounts: &A,
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
    crate::utils::create_account(
        accounts,
        accounts.core_message(),
        crate::sdk::compute_init_message_v1_space(payload.len()),
        &crate::ID,
        signer_seeds,
    )?;

    // Prepare (calling init and process instructions).
    crate::sdk::cpi::handle_prepare_message_v1(
        accounts.core_bridge_program(),
        accounts.core_message(),
        accounts.try_core_emitter_authority()?,
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
                message: accounts.core_message(),
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
    args: PostMessageArgs,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    match signer_seeds {
        Some(signer_seeds) => crate::legacy::cpi::post_message_unreliable(
            CpiContext::new_with_signer(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessageUnreliable {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: accounts.try_core_emitter()?,
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
                signer_seeds,
            ),
            args,
        ),
        None => crate::legacy::cpi::post_message_unreliable(
            CpiContext::new(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessageUnreliable {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: accounts.try_core_emitter()?,
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
            ),
            args,
        ),
    }
}

fn handle_prepared_message_v1<'info, A>(
    accounts: &A,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: PublishMessage<'info>,
{
    match signer_seeds {
        Some(signer_seeds) => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: None,
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
                signer_seeds,
            ),
            PostMessageArgs {
                nonce: 420, // not checked
                payload: Vec::new(),
                commitment: Commitment::Finalized, // not checked
            },
        ),
        None => crate::legacy::cpi::post_message(
            CpiContext::new(
                accounts.core_bridge_program(),
                crate::legacy::cpi::PostMessage {
                    config: accounts.core_bridge_config(),
                    message: accounts.core_message(),
                    emitter: None,
                    emitter_sequence: accounts.core_emitter_sequence(),
                    payer: accounts.payer(),
                    fee_collector: accounts.core_fee_collector(),
                    system_program: accounts.system_program(),
                },
            ),
            PostMessageArgs {
                nonce: 420, // not checked
                payload: Vec::new(),
                commitment: Commitment::Finalized, // not checked
            },
        ),
    }
}
