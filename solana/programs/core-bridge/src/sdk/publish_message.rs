use anchor_lang::prelude::*;

use super::{Commitment, PostMessageArgs};

/// Account context for invoking one of the ways you can publish a Wormhole message using the Core
/// Bridge program.
///
/// NOTE: A message's emitter address can either be a program's ID or a custom address (determined
/// by either a keypair or program's PDA). If the emitter address is a program ID, then
/// the seeds for the emitter authority must be \["emitter"\].
#[cfg(feature = "cpi")]
#[derive(Accounts)]
pub struct PublishMessage<'info> {
    /// Payer (mut signer).
    ///
    /// CHECK: This account's lamports will be used to create various accounts when publishing a
    /// Wormhole message.
    pub payer: AccountInfo<'info>,

    /// Core Bridge Message (mut).
    ///
    /// CHECK: This account can either be a generated keypair or an integrator's PDA. This message
    /// account can be closed only when it is in writing status.
    pub message: AccountInfo<'info>,

    /// Emitter authority (signer), who acts as the authority for crafting a Wormhole mesage.
    ///
    /// CHECK: This account is written to in the message that acts as the authority to continue
    /// writing to the message account. Once this authority is set, no one else can write and
    /// finalize this message.
    pub emitter_authority: AccountInfo<'info>,

    /// Core Bridge config account (mut).
    ///
    /// seeds: \["Bridge"\], seeds::program = core_bridge_program
    ///
    /// CHECK: This account is used to determine the Wormhole fee and the fee collector.
    pub config: AccountInfo<'info>,

    /// Emitter sequence tracker (mut).
    ///
    /// seeds: \["Sequence"\, emitter.key], seeds::program = core_bridge_program
    ///
    /// CHECK: This account is used to determine the sequence number for a particular emitter's
    /// message. It will be created if it does not exist.
    pub emitter_sequence: AccountInfo<'info>,

    /// Core Bridge Fee Collector (mut).
    ///
    /// seeds = \["fee_collector"\], seeds::program = core_bridge_program.
    ///
    /// CHECK: This system account is used to collect the Wormhole fee.
    pub fee_collector: Option<AccountInfo<'info>>,

    /// System program used to create accounts for this instruction.
    ///
    /// CHECK:
    pub system_program: AccountInfo<'info>,
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
#[cfg(feature = "cpi")]
pub fn publish_message<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, PublishMessage<'info>>,
    directive: PublishMessageDirective,
) -> Result<()> {
    match directive {
        PublishMessageDirective::Message {
            nonce,
            payload,
            commitment,
        } => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                ctx.program,
                crate::legacy::cpi::PostMessage {
                    config: ctx.accounts.config,
                    message: ctx.accounts.message,
                    emitter: Some(ctx.accounts.emitter_authority),
                    emitter_sequence: ctx.accounts.emitter_sequence,
                    payer: ctx.accounts.payer,
                    fee_collector: ctx.accounts.fee_collector,
                    system_program: ctx.accounts.system_program,
                },
                ctx.signer_seeds,
            ),
            PostMessageArgs {
                nonce,
                payload,
                commitment,
            },
        ),
        PublishMessageDirective::ProgramMessage {
            program_id,
            nonce,
            payload,
            commitment,
        } => handle_post_program_message_v1(ctx, program_id, nonce, payload, commitment),
        PublishMessageDirective::PreparedMessage => crate::legacy::cpi::post_message(
            CpiContext::new_with_signer(
                ctx.program,
                crate::legacy::cpi::PostMessage {
                    config: ctx.accounts.config,
                    message: ctx.accounts.message,
                    emitter: None,
                    emitter_sequence: ctx.accounts.emitter_sequence,
                    payer: ctx.accounts.payer,
                    fee_collector: ctx.accounts.fee_collector,
                    system_program: ctx.accounts.system_program,
                },
                ctx.signer_seeds,
            ),
            PostMessageArgs {
                nonce: 420, // not checked
                payload: Vec::new(),
                commitment: Commitment::Finalized, // not checked
            },
        ),
    }
}

fn handle_post_program_message_v1<'info>(
    ctx: CpiContext<'_, '_, '_, 'info, PublishMessage<'info>>,
    program_id: Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: Commitment,
) -> Result<()> {
    // Create message account.
    super::system_program::create_account_safe(
        CpiContext::new_with_signer(
            ctx.accounts.system_program.to_account_info(),
            super::system_program::CreateAccountSafe {
                payer: ctx.accounts.payer.to_account_info(),
                new_account: ctx.accounts.message.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        super::compute_prepared_message_space(payload.len()),
        &crate::ID,
    )?;

    // Prepare (calling init and process instructions).
    super::prepare_message(
        CpiContext::new_with_signer(
            ctx.program.to_account_info(),
            super::PrepareMessage {
                message: ctx.accounts.message.to_account_info(),
                emitter_authority: ctx.accounts.emitter_authority.to_account_info(),
            },
            ctx.signer_seeds,
        ),
        super::InitMessageV1Args {
            nonce,
            cpi_program_id: Some(program_id),
            commitment,
        },
        payload,
    )?;

    // Finally post.
    crate::legacy::cpi::post_message(
        CpiContext::new(
            ctx.program,
            crate::legacy::cpi::PostMessage {
                config: ctx.accounts.config,
                message: ctx.accounts.message,
                emitter: None,
                emitter_sequence: ctx.accounts.emitter_sequence,
                payer: ctx.accounts.payer,
                fee_collector: ctx.accounts.fee_collector,
                system_program: ctx.accounts.system_program,
            },
        ),
        PostMessageArgs {
            nonce: Default::default(), // not checked
            payload: Vec::new(),
            commitment: crate::types::Commitment::Finalized, // not checked
        },
    )
}
