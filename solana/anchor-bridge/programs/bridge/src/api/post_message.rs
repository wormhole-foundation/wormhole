use crate::{
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

type Message<'b> = Data<'b, PostedMessage, { AccountState::Uninitialized }>;

type Sequence<'b> = Data<'b, SequenceTracker, { AccountState::MaybeInitialized }>;

impl<'a, 'b: 'a> Seeded<&PostMessage<'b>> for Message<'b> {
    fn seeds(&self, accs: &PostMessage<'b>) -> Vec<Vec<u8>> {
        vec![
            accs.emitter.key.to_bytes().to_vec(),
            accs.sequence.sequence.to_be_bytes().to_vec(),
        ]
    }
}

impl<'b> Seeded<&PostMessage<'b>> for Sequence<'b> {
    fn seeds(&self, accs: &PostMessage<'b>) -> Vec<Vec<u8>> {
        vec![accs.emitter.key.to_bytes().to_vec()]
    }
}

pub type Bridge<'a> = Derive<Data<'a, BridgeData, { AccountState::Initialized }>, "Bridge">;
pub type FeeAccount<'a> = Derive<Info<'a>, "Fees">;

#[derive(FromAccounts)]
pub struct PostMessage<'b> {
    pub bridge: Bridge<'b>,
    pub fee_vault: FeeAccount<'b>,

    /// Account to store the posted message
    pub message: Message<'b>,

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
        self.message.verify_derivation(program_id, self)?;
        self.sequence.verify_derivation(program_id, self)?;
        Ok(())
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct PostMessageData {
    /// unique nonce for this message
    pub nonce: u32,
    /// message payload
    pub payload: Vec<u8>,
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

    // Init sequencce tracker if it does not exist yet.
    if !accs.sequence.is_initialized() {
        accs.sequence.create(accs, ctx, accs.payer.key, Exempt)?;
    }

    // Create message account
    accs.message.create(accs, ctx, accs.payer.key, Exempt)?;

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
