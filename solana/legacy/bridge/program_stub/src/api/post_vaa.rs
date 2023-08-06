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
        PostedVAA,
        PostedVAADerivationData,
    },
    instructions::hash_vaa,
    PostVAAData,
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};

#[derive(FromAccounts)]
pub struct PostVAA<'b> {
    /// Information about the current guardian set.
    pub guardian_set: Info<'b>,

    /// Bridge Info
    pub bridge_info: Bridge<'b, { AccountState::Initialized }>,

    /// Signature Info
    pub signature_set: Info<'b>,

    /// Message the VAA is associated with.
    pub message: Mut<PostedVAA<'b, { AccountState::MaybeInitialized }>>,

    /// Account used to pay for auxillary instructions.
    pub payer: Mut<Signer<Info<'b>>>,

    /// Clock used for timestamping.
    pub clock: Sysvar<'b, Clock>,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct Signature {
    pub index: u8,
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
}

pub type ForeignAddress = [u8; 32];

pub fn post_vaa(ctx: &ExecutionContext, accs: &mut PostVAA, vaa: PostVAAData) -> Result<()> {
    let msg_derivation = PostedVAADerivationData {
        payload_hash: hash_vaa(&vaa).to_vec(),
    };

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
