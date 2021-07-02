use crate::{
    accounts::{
        Bridge,
        FeeCollector,
        Message,
        MessageDerivationData,
        Sequence,
        SequenceDerivationData,
    },
    Error::{
        InsufficientFees,
        MathOverflow,
    },
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

pub type UninitializedMessage<'b> = Message<'b, { AccountState::Uninitialized }>;

impl<'a> From<&PostMessage<'a>> for SequenceDerivationData<'a> {
    fn from(accs: &PostMessage<'a>) -> Self {
        SequenceDerivationData {
            emitter_key: accs.emitter.key,
        }
    }
}

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    pub bridge: Bridge<'b, { AccountState::Initialized }>,

    /// Account to store the posted message
    pub message: UninitializedMessage<'b>,

    /// Emitter of the VAA
    pub emitter: Signer<Info<'b>>,

    /// Tracker for the emitter sequence
    pub sequence: Sequence<'b>,

    /// Payer for account creation
    pub payer: Signer<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: FeeCollector<'b>,

    pub clock: Sysvar<'b, Clock>,
}

impl<'b> InstructionContext<'b> for PostMessage<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        self.sequence.verify_derivation(program_id, &self.into())?;
        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PostMessageData {
    /// Unique nonce for this message
    pub nonce: u32,

    /// Message payload
    pub payload: Vec<u8>,

    /// Should the VAA for this message be persisted on-chain
    pub persist: bool,
}

pub fn post_message(
    ctx: &ExecutionContext,
    accs: &mut PostMessage,
    data: PostMessageData,
) -> Result<()> {
    trace!("Message Address: {}", accs.message.info().key);
    trace!("Emitter Address: {}", accs.emitter.info().key);
    trace!("Nonce: {}", data.nonce);

    let msg_derivation = MessageDerivationData {
        emitter_key: accs.emitter.key.to_bytes(),
        emitter_chain: CHAIN_ID_SOLANA,
        nonce: data.nonce,
        payload: data.payload.clone(),
        sequence: None,
    };

    accs.message
        .verify_derivation(ctx.program_id, &msg_derivation)?;

    // Fee handling, checking previously known balance allows us to not care who is the payer of
    // this submission.
    if accs
        .fee_collector
        .lamports()
        .checked_sub(accs.bridge.last_lamports)
        .ok_or(MathOverflow)?
        < accs.bridge.config.fee
    {
        trace!(
            "Expected fee not found: fee, last_lamports, collector: {} {} {}",
            accs.bridge.config.fee,
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

    // Initialize transfer
    trace!("Setting Message Details");
    accs.message.submission_time = accs.clock.unix_timestamp as u32;
    accs.message.emitter_chain = CHAIN_ID_SOLANA;
    accs.message.emitter_address = accs.emitter.key.to_bytes();
    accs.message.nonce = data.nonce;
    accs.message.payload = data.payload;
    accs.message.sequence = accs.sequence.sequence;
    accs.message.persist = data.persist;

    // Create message account
    accs.message
        .create(&msg_derivation, ctx, accs.payer.key, Exempt)?;

    // Bump sequence number
    trace!("New Sequence: {}", accs.sequence.sequence + 1);
    accs.sequence.sequence += 1;

    Ok(())
}
