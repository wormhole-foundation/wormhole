use solitaire::*;

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use sha3::Digest;
use solana_program::{
    self,
    sysvar::clock::Clock,
};
use std::io::{
    Cursor,
    Write,
};

use crate::{
    types::{
        self,
        Bridge,
    },
    Error,
    VAA_TX_FEE,
};

const MIN_BRIDGE_BALANCE: u64 = (((solana_program::rent::ACCOUNT_STORAGE_OVERHEAD
    + std::mem::size_of::<Bridge>() as u64)
    * solana_program::rent::DEFAULT_LAMPORTS_PER_BYTE_YEAR) as f64
    * solana_program::rent::DEFAULT_EXEMPTION_THRESHOLD) as u64;

type GuardianSet<'b> = Derive<Data<'b, types::GuardianSet>, "GuardianSet">;
type SignatureSet<'b> = Derive<Data<'b, types::SignatureSet>, "Signatures">;
type Message<'b> = Derive<Data<'b, types::PostedMessage>, "Message">;

#[derive(FromAccounts)]
pub struct PostVAA<'b> {
    /// Required by Anchor for associated accounts.
    pub system_program: Info<'b>,

    /// Required by Anchor for associated accounts.
    pub rent: Info<'b>,

    /// Clock used for timestamping.
    pub clock: Sysvar<Info<'b>, Clock>,

    /// State struct, derived by #[state], used for associated accounts.
    pub state: Info<'b>,

    /// Information about the current guardian set.
    pub guardian_set: GuardianSet<'b>,

    /// Bridge Info
    pub bridge_info: Info<'b>,

    /// Claim Info
    pub claim: Info<'b>,

    /// Signature Info
    pub signature_set: SignatureSet<'b>,

    /// Account used to pay for auxillary instructions.
    pub payer: Info<'b>,

    /// Message the VAA is associated with.
    pub message: Message<'b>,
}

impl<'b> InstructionContext<'b> for PostVAA<'b> {
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
    pub signatures: Vec<Signature>,

    // Body part
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u8,
    pub emitter_address: ForeignAddress,
    pub payload: Vec<u8>,
}

pub fn post_vaa(ctx: &ExecutionContext, accs: &mut PostVAA, vaa: PostVAAData) -> Result<()> {
    // Verify any required invariants before we process the instruction.
    check_active(&accs.guardian_set, &accs.clock)?;
    check_valid_sigs(&accs.guardian_set, &accs.signature_set)?;
    check_integrity(&vaa, &accs.signature_set)?;

    // Count the numnber of signatures currently present.
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
        // Division by 10+1 to bring the number back into range.
        len / (10 + 1)
    };

    if signature_count < required_consensus_count {
        return Err(Error::PostVAAConsensusFailed.into());
    }

    // Store VAA data in associated message.
    accs.message.vaa_version = vaa.version;
    accs.message.vaa_time = vaa.timestamp;
    /* accs.message.vaa_signature_account = */
    *accs.signature_set.info().key;

    // If the bridge has enough balance, refund the SOL to the transaction payer.
    // if VAA_TX_FEE + MIN_BRIDGE_BALANCE < accs.state.info().lamports() {
    //     transfer_sol(
    //         &accs.state.info(),
    //         &accs.payer,
    //         VAA_TX_FEE,
    //     )?;
    // }
    ////
    //    // Claim the VAA
    //    ctx.accounts.claim.vaa_time = ctx.accounts.clock.unix_timestamp as u32;
    Ok(())
}

fn transfer_sol(sender: &Info, recipient: &Info, amount: u64) -> Result<()> {
    //    let mut payer_balance = sender.try_borrow_mut_lamports()?;
    //    **payer_balance = payer_balance
    //        .checked_sub(amount)
    //        .ok_or(ProgramError::InsufficientFunds)?;
    //    let mut recipient_balance = recipient.try_borrow_mut_lamports()?;
    //    **recipient_balance = recipient_balance
    //        .checked_add(amount)
    //        .ok_or(ProgramError::InvalidArgument)?;
    Ok(())
}

/// A guardian set must not have expired.
#[inline(always)]
fn check_active<'r>(guardian_set: &GuardianSet, clock: &Sysvar<Info<'r>, Clock>) -> Result<()> {
    //    if guardian_set.expiration_time != 0
    //        && (guardian_set.expiration_time as i64) < clock.unix_timestamp
    //    {
    //        return Err(ErrorCode::PostVAAGuardianSetExpired.into());
    //    }
    Ok(())
}

/// The signatures in this instruction must be from the right guardian set.
#[inline(always)]
fn check_valid_sigs<'r>(guardian_set: &GuardianSet, signatures: &SignatureSet<'r>) -> Result<()> {
    //    if sig_info.guardian_set_index != guardian_set.index {
    //        return Err(ErrorCode::PostVAAGuardianSetMismatch.into());
    //    }
    Ok(())
}

#[inline(always)]
fn check_integrity<'r>(vaa: &PostVAAData, signatures: &SignatureSet<'r>) -> Result<()> {
    //    // Serialize the VAA body into an array of bytes.
    //    let body = {
    //        let mut v = Cursor::new(Vec::new());
    //        v.write_u32::<BigEndian>(vaa.timestamp)?;
    //        v.write_u32::<BigEndian>(vaa.nonce)?;
    //        v.write_u8(vaa.emitter_chain)?;
    //        v.write(&vaa.emitter_address)?;
    //        v.write(&vaa.payload)?;
    //        v.into_inner()
    //    };
    //    // Hash this body, which is expected to be the same as the hash currently stored in the
    //    // signature account, binding that set of signatures to this VAA.
    //    let body_hash: [u8; 32] = {
    //        let mut h = sha3::Keccak256::default();
    //        h.write(body.as_slice())
    //            .map_err(|_| ProgramError::InvalidArgument);
    //        h.finalize().into()
    //    };
    //    if signatures.hash != body_hash {
    //        return Err(ProgramError::InvalidAccountData.into());
    //    }
    Ok(())
}
