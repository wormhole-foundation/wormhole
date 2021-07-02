use solitaire::*;

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::{
    self,
    sysvar::clock::Clock,
};

use crate::{
    accounts::{
        Bridge,
        GuardianSet,
        GuardianSetDerivationData,
        Message,
        MessageDerivationData,
        SignatureSet,
        SignatureSetDerivationData,
    },
    Error,
    Error::GuardianSetMismatch,
    CHAIN_ID_SOLANA,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use sha3::Digest;
use solana_program::{
    program_error::ProgramError,
    pubkey::Pubkey,
};
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};
use std::io::{
    Cursor,
    Write,
};

impl<'a> From<&PostVAA<'a>> for SignatureSetDerivationData {
    fn from(accs: &PostVAA<'a>) -> Self {
        SignatureSetDerivationData {
            hash: accs.signature_set.hash,
        }
    }
}

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
    pub guardian_set: GuardianSet<'b, { AccountState::Initialized }>,

    /// Bridge Info
    pub bridge_info: Bridge<'b, { AccountState::Initialized }>,

    /// Signature Info
    pub signature_set: SignatureSet<'b, { AccountState::Initialized }>,

    /// Message the VAA is associated with.
    pub message: Message<'b, { AccountState::MaybeInitialized }>,

    /// Account used to pay for auxillary instructions.
    pub payer: Info<'b>,

    /// Clock used for timestamping.
    pub clock: Sysvar<'b, Clock>,
}

impl<'b> InstructionContext<'b> for PostVAA<'b> {
    fn verify(&self, program_id: &Pubkey) -> Result<()> {
        self.signature_set
            .verify_derivation(program_id, &self.into())?;
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
    accs.guardian_set
        .verify_derivation(ctx.program_id, &(&vaa).into())?;

    // Verify any required invariants before we process the instruction.
    check_active(&accs.guardian_set, &accs.clock)?;
    check_valid_sigs(&accs.guardian_set, &accs.signature_set)?;
    check_integrity(&vaa, &accs.signature_set)?;

    // Count the number of signatures currently present.
    let signature_count: usize = accs
        .signature_set
        .signatures
        .iter()
        .filter(|v| v.iter().filter(|v| **v != 0).count() != 0)
        .count();

    // Calculate how many signatures are required to reach consensus. This calculation is in
    // expanded form to ease auditing.
    let required_consensus_count = {
        let len = accs.guardian_set.keys.len();
        // Fixed point number transformation with one decimal to deal with rounding.
        let len = (len * 10) / 3;
        // Multiplication by two to get a 2/3 quorum.
        let len = len * 2;
        // Division to bring number back into range.
        len / 10 + 1
    };

    if signature_count < required_consensus_count {
        return Err(Error::PostVAAConsensusFailed.into());
    }

    // If the VAA originates from another chain we need to create the account and populate all fields
    if vaa.emitter_chain != CHAIN_ID_SOLANA {
        accs.message.nonce = vaa.nonce;
        accs.message.emitter_chain = vaa.emitter_chain;
        accs.message.emitter_address = vaa.emitter_address;
        accs.message.sequence = vaa.sequence;
        accs.message.payload = vaa.payload;
        accs.message
            .create(&msg_derivation, ctx, accs.payer.key, Exempt)?;
    }

    // Store VAA data in associated message.
    accs.message.vaa_version = vaa.version;
    accs.message.vaa_time = vaa.timestamp;
    accs.message.vaa_signature_account = *accs.signature_set.info().key;

    Ok(())
}

/// A guardian set must not have expired.
#[inline(always)]
fn check_active<'r>(
    guardian_set: &GuardianSet<'r, { AccountState::Initialized }>,
    clock: &Sysvar<'r, Clock>,
) -> Result<()> {
    if guardian_set.expiration_time != 0
        && (guardian_set.expiration_time as i64) < clock.unix_timestamp
    {
        return Err(Error::PostVAAGuardianSetExpired.into());
    }
    Ok(())
}

/// The signatures in this instruction must be from the right guardian set.
#[inline(always)]
fn check_valid_sigs<'r>(
    guardian_set: &GuardianSet<'r, { AccountState::Initialized }>,
    signatures: &SignatureSet<'r, { AccountState::Initialized }>,
) -> Result<()> {
    if signatures.guardian_set_index != guardian_set.index {
        return Err(GuardianSetMismatch.into());
    }
    Ok(())
}

#[inline(always)]
fn check_integrity<'r>(
    vaa: &PostVAAData,
    signatures: &SignatureSet<'r, { AccountState::Initialized }>,
) -> Result<()> {
    // Serialize the VAA body into an array of bytes.
    let body = {
        let mut v = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(vaa.timestamp)?;
        v.write_u32::<BigEndian>(vaa.nonce)?;
        v.write_u16::<BigEndian>(vaa.emitter_chain)?;
        v.write(&vaa.emitter_address)?;
        v.write_u64::<BigEndian>(vaa.sequence)?;
        v.write(&vaa.payload)?;
        v.into_inner()
    };

    // Hash this body, which is expected to be the same as the hash currently stored in the
    // signature account, binding that set of signatures to this VAA.
    let body_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write(body.as_slice())
            .map_err(|_| ProgramError::InvalidArgument)?;
        h.finalize().into()
    };

    if signatures.hash != body_hash {
        return Err(ProgramError::InvalidAccountData.into());
    }
    Ok(())
}
