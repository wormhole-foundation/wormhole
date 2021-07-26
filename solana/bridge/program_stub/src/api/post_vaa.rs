use solitaire::*;

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::{
    self,
    sysvar::clock::Clock,
};

use bridge::{
    accounts::{
        Bridge,
        GuardianSetDerivationData,
        Message,
        MessageDerivationData,
    },
    CHAIN_ID_SOLANA,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};

impl From<&PostVAAData> for GuardianSetDerivationData {
    fn from(data: &PostVAAData) -> Self {
        GuardianSetDerivationData {
            index: data.guardian_set_index,
        }
    }
}

#[derive(FromAccounts)]
pub struct PostVAA<'b> {
    /// Information about the current guardian set.
    pub guardian_set: Info<'b>,

    /// Bridge Info
    pub bridge_info: Bridge<'b, { AccountState::Initialized }>,

    /// Signature Info
    pub signature_set: Info<'b>,

    /// Message the VAA is associated with.
    pub message: Mut<Message<'b, { AccountState::MaybeInitialized }>>,

    /// Account used to pay for auxillary instructions.
    pub payer: Mut<Signer<Info<'b>>>,

    /// Clock used for timestamping.
    pub clock: Sysvar<'b, Clock>,
}

impl<'b> InstructionContext<'b> for PostVAA<'b> {
    fn verify(&self, _program_id: &Pubkey) -> Result<()> {
        Ok(())
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct Signature {
    pub index: u8,
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
}

pub type ForeignAddress = [u8; 32];

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct PostVAAData {
    // Header part
    pub version: u8,
    pub guardian_set_index: u32,

    // Body part
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u16,
    pub emitter_address: ForeignAddress,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: Vec<u8>,
}

pub fn post_vaa(ctx: &ExecutionContext, accs: &mut PostVAA, vaa: PostVAAData) -> Result<()> {
    let mut msg_derivation = MessageDerivationData {
        emitter_key: vaa.emitter_address,
        emitter_chain: vaa.emitter_chain,
        nonce: vaa.nonce,
        payload: vaa.payload.clone(),
        sequence: None,
    };
    if vaa.emitter_chain != CHAIN_ID_SOLANA {
        msg_derivation.sequence = Some(vaa.sequence)
    }

    accs.message
        .verify_derivation(ctx.program_id, &msg_derivation)?;

    // If the VAA originates from another chain we need to create the account and populate all fields
    if !accs.message.is_initialized() {
        accs.message.nonce = vaa.nonce;
        accs.message.emitter_chain = vaa.emitter_chain;
        accs.message.emitter_address = vaa.emitter_address;
        accs.message.sequence = vaa.sequence;
        accs.message.payload = vaa.payload;
        accs.message.consistency_level = vaa.consistency_level;
        accs.message
            .create(&msg_derivation, ctx, accs.payer.key, Exempt)?;
    }

    // Store VAA data in associated message.
    accs.message.vaa_version = vaa.version;
    accs.message.vaa_time = vaa.timestamp;
    accs.message.vaa_signature_account = *accs.signature_set.info().key;

    Ok(())
}
