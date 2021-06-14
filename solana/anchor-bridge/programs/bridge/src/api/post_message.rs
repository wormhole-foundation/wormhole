use crate::{
    accounts::{
        Bridge,
        Message,
        MessageDerivationData,
        Sequence,
        SequenceDerivationData,
    },
    types::{
        BridgeData,
        PostedMessage,
        SequenceTracker,
    },
    Error::{
        InsufficientFees,
        MathOverflow,
    },
};
use solana_program::{
    pubkey::Pubkey,
    sysvar::clock::Clock,
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
    *,
};

type UninitializedMessage<'b> = Message<'b, { AccountState::Uninitialized }>;

impl<'a> From<&PostMessage<'a>> for MessageDerivationData {
    fn from(accs: &PostMessage<'a>) -> Self {
        MessageDerivationData {
            emitter_key: accs.emitter.key.to_bytes(),
            sequence: accs.sequence.sequence,
        }
    }
}

impl<'a> From<&PostMessage<'a>> for SequenceDerivationData<'a> {
    fn from(accs: &PostMessage<'a>) -> Self {
        SequenceDerivationData {
            emitter_key: accs.emitter.key,
        }
    }
}

pub type FeeAccount<'a> = Derive<Info<'a>, "Fees">;

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    pub bridge: Bridge<'b, { AccountState::Initialized }>,
    pub fee_vault: FeeAccount<'b>,

    /// Account to store the posted message
    pub message: UninitializedMessage<'b>,

    /// Emitter of the VAA
    pub emitter: Signer<Info<'b>>,

    /// Tracker for the emitter sequence
    pub sequence: Sequence<'b>,

    /// Payer for account creation
    pub payer: Signer<Info<'b>>,

    /// Account to collect tx fee
    pub fee_collector: Derive<Info<'b>, "fee_collector">,

    /// Instruction reflection account (special sysvar)
    pub instruction_acc: Info<'b>,

    pub clock: Sysvar<'b, Clock>,
}

impl<'b> InstructionContext<'b> for PostMessage<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        self.message.verify_derivation(program_id, &self.into())?;
        self.sequence.verify_derivation(program_id, &self.into())?;
        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PostMessageData {
    /// unique nonce for this message
    pub nonce: u32,
    /// message payload
    pub payload: Vec<u8>,
    /// Emitter address
    pub emitter: Pubkey,
}

pub fn post_message(
    ctx: &ExecutionContext,
    accs: &mut PostMessage,
    data: PostMessageData,
) -> Result<()> {
    // Fee handling
    let fee = transfer_fee();
    if accs
        .fee_collector
        .lamports()
        .checked_sub(accs.bridge.last_lamports)
        .ok_or(MathOverflow)?
        < fee
    {
        return Err(InsufficientFees.into());
    }
    accs.bridge.last_lamports = accs.fee_collector.lamports();

    // Init sequence tracker if it does not exist yet.
    if !accs.sequence.is_initialized() {
        accs.sequence
            .create(&(&*accs).into(), ctx, accs.payer.key, Exempt)?;
    }

    // Create message account
    accs.message
        .create(&(&*accs).into(), ctx, accs.payer.key, Exempt)?;

    // Initialize transfer
    accs.message.submission_time = accs.clock.unix_timestamp as u32;
    accs.message.emitter_chain = 1;
    accs.message.emitter_address = accs.emitter.key.to_bytes();
    accs.message.nonce = data.nonce;
    accs.message.payload = data.payload.clone();
    accs.message.sequence = accs.sequence.sequence;

    // Bump sequence number
    accs.sequence.sequence += 1;

    Ok(())
}

pub fn transfer_fee() -> u64 {
    500
}
