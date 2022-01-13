use crate::{
    accounts::{
        Bridge,
        FeeCollector,
        PostedMessage,
        Sequence,
        SequenceDerivationData,
    },
    error::Error::{
        InsufficientFees,
        MathOverflow,
    },
    types::ConsistencyLevel,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    log::sol_log_data,
    msg,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::Seeded,
    trace,
    CreationLamports::Exempt,
    *,
};

pub type UninitializedMessage<'b> = PostedMessage<'b, { AccountState::Uninitialized }>;

impl<'a> From<&PostMessage<'a>> for SequenceDerivationData<'a> {
    fn from(accs: &PostMessage<'a>) -> Self {
        SequenceDerivationData {
            emitter_key: accs.emitter.key,
        }
    }
}

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    /// Bridge config needed for fee calculation.
    pub bridge: Mut<Bridge<'b, { AccountState::Initialized }>>,

    /// Account to store the posted message if persistence is requested
    pub message: Signer<MaybeMut<UninitializedMessage<'b>>>,

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

impl<'b> InstructionContext<'b> for PostMessage<'b> {
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
    trace!("Message Address: {}", accs.message.info().key);
    trace!("Emitter Address: {}", accs.emitter.info().key);
    trace!("Nonce: {}", data.nonce);

    accs.sequence
        .verify_derivation(ctx.program_id, &(&*accs).into())?;

    let fee = accs.bridge.config.fee;
    // Fee handling, checking previously known balance allows us to not care who is the payer of
    // this submission.
    if accs
        .fee_collector
        .lamports()
        .checked_sub(accs.bridge.last_lamports)
        .ok_or(MathOverflow)?
        < fee
    {
        trace!(
            "Expected fee not found: fee, last_lamports, collector: {} {} {}",
            fee,
            accs.bridge.last_lamports,
            accs.fee_collector.lamports(),
        );
        return Err(InsufficientFees.into());
    }
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    // Init sequence tracker if it does not exist yet.
    if !accs.sequence.is_initialized() {
        trace!("Initializing Sequence account to 0.");
        accs.sequence
            .create(&(&*accs).into(), ctx, accs.payer.key, Exempt)?;
    }

    // DO NOT REMOVE - CRITICAL OUTPUT
    msg!("Sequence: {}", accs.sequence.sequence);

    // Initialize transfer
    trace!("Setting Message Details");
    accs.message.submission_time = accs.clock.unix_timestamp as u32;
    accs.message.emitter_chain = CHAIN_ID_SOLANA;
    accs.message.emitter_address = accs.emitter.key.to_bytes();
    accs.message.nonce = data.nonce;
    accs.message.payload = data.payload;
    accs.message.sequence = accs.sequence.sequence;
    accs.message.consistency_level = match data.consistency_level {
        ConsistencyLevel::Confirmed => 1,
        ConsistencyLevel::Finalized => 32,
    };

    // Log the message data
    sol_log_data(&[accs.message.try_to_vec()?.as_slice()]);

    if accs.message.info().is_writable {
        // Create message account
        let size = accs.message.size();
        let ix = solana_program::system_instruction::create_account(
            accs.payer.key,
            accs.message.info().key,
            Exempt.amount(size),
            size as u64,
            ctx.program_id,
        );
        solana_program::program::invoke(&ix, ctx.accounts)?;
    }

    // Bump sequence number
    trace!("New Sequence: {}", accs.sequence.sequence + 1);
    accs.sequence.sequence += 1;

    Ok(())
}
