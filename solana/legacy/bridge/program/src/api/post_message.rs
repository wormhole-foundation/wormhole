use crate::{
    accounts::{
        Bridge,
        FeeCollector,
        PostedMessage,
        PostedMessageUnreliable,
        Sequence,
        SequenceDerivationData,
    },
    error::Error::{
        EmitterChanged,
        InsufficientFees,
        InvalidPayloadLength,
        MathOverflow,
    },
    types::ConsistencyLevel,
    IsSigned::*,
    MessageData,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    msg,
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::Seeded,
    trace,
    CreationLamports::Exempt,
    *,
};

pub type UninitializedMessage<'b> = PostedMessage<'b, { AccountState::Uninitialized }>;

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    /// Bridge config needed for fee calculation.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<UninitializedMessage<'b>>>,

    /// Emitter of the VAA
    pub emitter: Signer<MaybeMut<Info<'b>>>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Sequence<'b>>,

    /// Payer for account creation
    pub payer: Mut<Signer<Info<'b>>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<FeeCollector<'b>>,

    pub clock: Sysvar<'b, Clock>,
}

#[derive(FromAccounts)]
pub struct PostMessageUnreliable<'b> {
    /// Bridge config needed for fee calculation.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message
    pub message: Signer<Mut<PostedMessageUnreliable<'b, { AccountState::MaybeInitialized }>>>,

    /// Emitter of the VAA
    pub emitter: Signer<MaybeMut<Info<'b>>>,

    /// Tracker for the emitter sequence
    pub sequence: Mut<Sequence<'b>>,

    /// Payer for account creation
    pub payer: Mut<Signer<Info<'b>>>,

    /// Account to collect tx fee
    pub fee_collector: Mut<FeeCollector<'b>>,

    pub clock: Sysvar<'b, Clock>,
}

#[derive(BorshDeserialize, BorshSerialize)]
pub struct PostMessageData {
    /// Unique nonce for this message
    pub nonce: u32,

    /// Message payload
    pub payload: Vec<u8>,

    /// Commitment Level required for an attestation to be produced
    pub consistency_level: ConsistencyLevel,
}

pub fn post_message(
    ctx: &ExecutionContext,
    accs: &mut PostMessage,
    data: PostMessageData,
) -> Result<()> {
    post_message_internal(
        ctx,
        &mut accs.bridge,
        accs.message.info().key,
        &mut accs.message,
        &mut accs.emitter,
        &mut accs.sequence,
        &mut accs.payer,
        &mut accs.fee_collector,
        &mut accs.clock,
        data,
    )?;

    // Create message account
    let size = accs.message.size();
    create_account(
        ctx,
        accs.message.info(),
        accs.payer.key,
        Exempt,
        size,
        ctx.program_id,
        NotSigned,
    )?;

    Ok(())
}

/// Post a message while reusing the message account. This saves the rent that would be required for
/// allocating a new message account. When an account is reused and the guardians don't pick up the
/// message due to network instability or a bug there is NO way to recover the message if it has
/// been overwritten. This makes this instruction useful for use-cases that require high number of
/// messages to be published but don't require 100% delivery guarantee.
/// DO NOT USE THIS FOR USE-CASES THAT MOVE VALUE; MESSAGES MAY NOT BE DELIVERED
pub fn post_message_unreliable(
    ctx: &ExecutionContext,
    accs: &mut PostMessageUnreliable,
    data: PostMessageData,
) -> Result<()> {
    // Accounts can't be resized so the payload sizes need to match
    if accs.message.is_initialized() && accs.message.payload.len() != data.payload.len() {
        return Err(InvalidPayloadLength.into());
    }
    // The emitter must be identical
    if accs.message.is_initialized() && accs.emitter.key.to_bytes() != accs.message.emitter_address
    {
        return Err(EmitterChanged.into());
    }

    post_message_internal(
        ctx,
        &mut accs.bridge,
        accs.message.info().key,
        &mut accs.message,
        &mut accs.emitter,
        &mut accs.sequence,
        &mut accs.payer,
        &mut accs.fee_collector,
        &mut accs.clock,
        data,
    )?;

    if !accs.message.is_initialized() {
        // Create message account
        let size = accs.message.size();
        create_account(
            ctx,
            accs.message.info(),
            accs.payer.key,
            Exempt,
            size,
            ctx.program_id,
            NotSigned,
        )?;
    }

    Ok(())
}

#[allow(clippy::too_many_arguments)]
fn post_message_internal<'b>(
    ctx: &ExecutionContext,
    bridge: &mut Mut<Bridge<'b, { AccountState::Initialized }>>,
    #[cfg_attr(not(feature = "trace"), allow(unused_variables))] message_key: &Pubkey,
    message: &mut MessageData,
    emitter: &mut Signer<MaybeMut<Info<'b>>>,
    sequence: &mut Mut<Sequence<'b>>,
    payer: &mut Mut<Signer<Info<'b>>>,
    fee_collector: &mut Mut<FeeCollector<'b>>,
    clock: &mut Sysvar<'b, Clock>,
    data: PostMessageData,
) -> Result<()> {
    trace!("Message Address: {}", message_key);
    trace!("Emitter Address: {}", emitter.info().key);
    trace!("Nonce: {}", data.nonce);

    let sequence_derivation = SequenceDerivationData {
        emitter_key: emitter.key,
    };
    sequence.verify_derivation(ctx.program_id, &sequence_derivation)?;

    let fee = bridge.config.fee;
    // Fee handling, checking previously known balance allows us to not care who is the payer of
    // this submission.
    if fee_collector
        .lamports()
        .checked_sub(bridge.last_lamports)
        .ok_or(MathOverflow)?
        < fee
    {
        trace!(
            "Expected fee not found: fee, last_lamports, collector: {} {} {}",
            fee,
            bridge.last_lamports,
            fee_collector.lamports(),
        );
        return Err(InsufficientFees.into());
    }
    bridge.last_lamports = fee_collector.lamports();

    // Init sequence tracker if it does not exist yet.
    if !sequence.is_initialized() {
        trace!("Initializing Sequence account to 0.");
        sequence.create(&sequence_derivation, ctx, payer.key, Exempt)?;
    }

    // DO NOT REMOVE - CRITICAL OUTPUT
    msg!("Sequence: {}", sequence.sequence);

    // Initialize transfer
    trace!("Setting Message Details");
    message.submission_time = clock.unix_timestamp as u32;
    message.emitter_chain = CHAIN_ID_SOLANA;
    message.emitter_address = emitter.key.to_bytes();
    message.nonce = data.nonce;
    message.payload = data.payload;
    message.sequence = sequence.sequence;
    message.consistency_level = match data.consistency_level {
        ConsistencyLevel::Confirmed => 1,
        ConsistencyLevel::Finalized => 32,
    };

    // Bump sequence number
    trace!("New Sequence: {}", sequence.sequence + 1);
    sequence.sequence += 1;

    Ok(())
}
