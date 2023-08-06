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
        PostedVAA,
        PostedVAADerivationData,
        SignatureSet,
    },
    error::Error::{
        GuardianSetMismatch,
        PostVAAConsensusFailed,
        PostVAAGuardianSetExpired,
        VAAInvalid,
    },
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use serde::{
    Deserialize,
    Serialize,
};
use sha3::Digest;
use solana_program::program_error::ProgramError;
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};
use std::io::{
    Cursor,
    Write,
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
    pub guardian_set: GuardianSet<'b, { AccountState::Initialized }>,

    /// Bridge Info
    pub bridge_info: Bridge<'b, { AccountState::Initialized }>,

    /// Signature Info
    pub signature_set: SignatureSet<'b, { AccountState::Initialized }>,

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

#[derive(Default, BorshSerialize, BorshDeserialize, Clone, Serialize, Deserialize)]
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
    let msg_derivation = PostedVAADerivationData {
        payload_hash: accs.signature_set.hash.to_vec(),
    };

    accs.message
        .verify_derivation(ctx.program_id, &msg_derivation)?;
    accs.guardian_set
        .verify_derivation(ctx.program_id, &(&vaa).into())?;

    if accs.message.is_initialized() {
        return Ok(());
    }

    // Verify any required invariants before we process the instruction.
    check_active(&accs.guardian_set, &accs.clock)?;
    check_valid_sigs(&accs.guardian_set, &accs.signature_set)?;
    check_integrity(&vaa, &accs.signature_set)?;

    // Count the number of signatures currently present.
    let signature_count: usize = accs.signature_set.signatures.iter().filter(|v| **v).count();

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
        return Err(PostVAAConsensusFailed.into());
    }

    // Persist VAA data
    accs.message.nonce = vaa.nonce;
    accs.message.emitter_chain = vaa.emitter_chain;
    accs.message.emitter_address = vaa.emitter_address;
    accs.message.sequence = vaa.sequence;
    accs.message.payload = vaa.payload;
    accs.message.consistency_level = vaa.consistency_level;
    accs.message.vaa_version = vaa.version;
    accs.message.vaa_time = vaa.timestamp;
    accs.message.vaa_signature_account = *accs.signature_set.info().key;
    accs.message
        .create(&msg_derivation, ctx, accs.payer.key, Exempt)?;

    Ok(())
}

/// A guardian set must not have expired.
#[inline(always)]
fn check_active<'r>(
    guardian_set: &GuardianSet<'r, { AccountState::Initialized }>,
    clock: &Sysvar<'r, Clock>,
) -> Result<()> {
    // IMPORTANT - this is a fix for mainnet wormhole
    // The initial guardian set was never expired so we block it here.
    if guardian_set.index == 0 && guardian_set.creation_time == 1628099186 {
        return Err(PostVAAGuardianSetExpired.into());
    }
    if guardian_set.expiration_time != 0
        && (guardian_set.expiration_time as i64) < clock.unix_timestamp
    {
        return Err(PostVAAGuardianSetExpired.into());
    }
    Ok(())
}

// Static list of invalid signature accounts that are not allowed to post VAAs.
static INVALID_SIGNATURES: &[&str; 16] = &[
    "18eK1799CaNMGCUnnCt1Kq2uwKkax6T2WmtrDsZuVFQ",
    "2g6NCUUPaD6AxdHPQMVLpjpAvBfKMek6dDiGUe2A6T33",
    "3hYV5968hNzbqUfcvnQ6v9D5h32hEwGJn19c47N3unNj",
    "76eEyhaEKs4mesjiQiu8bghvwDHNxJW3EfcpbNC78y1z",
    "7PdcxSn7xk2UN5VYmKnJ2Q64PdBhbBQFf4RwHqhQCMgv",
    "94wXN3z3Pph2vMVaviZSouo7oCDqt4fekvqT3FYJSrWA",
    "AXe9VXd9jjXkBxSdvgj4bHSZNeqxY73sSQEsp1tnekY4",
    "B2hS49B8n4Ad6cxZLoAjz7Hux7Kf17D5xUX3neDPHpug",
    "BTXnYYjnfXByqJprarqzp65Yha2XwQVmg8V8KWBhr6aA",
    "Bzb5G4Y8QcaMVMQq3r8q1SuKSxtgnWSFdKCEisJCbcBP",
    "CJfRUQxyonG6B5mnztsNUqxknbFT89DJdrdrzV9F96mU",
    "CK1j9TxWP1T5w1QzFu4vPDAbUR34mfVqvk5wziE8TzST",
    "E8qKJMwzBCiHCHUmBEcL631kN5CjfsHNx24osFLfHg69",
    "EtMw1nQ4AQaH53RjYz3pRk12rrqWjcYjPDETphYJzmCX",
    "EVNwqfgkUnJoMqBqiHgDfa3TLZPQocX1hpcbAXbpcSLv",
    "FixSiDfTxvoy5Zgjp5KdFU8U23ChwCxPWY3WTkmMW2fU",
];

/// The signatures in this instruction must be from the right guardian set.
#[inline(always)]
fn check_valid_sigs<'r>(
    guardian_set: &GuardianSet<'r, { AccountState::Initialized }>,
    signatures: &SignatureSet<'r, { AccountState::Initialized }>,
) -> Result<()> {
    if signatures.guardian_set_index != guardian_set.index {
        return Err(GuardianSetMismatch.into());
    }

    // Reject blacklisted signature accounts.
    if INVALID_SIGNATURES.contains(&&*signatures.info().key.to_string()) {
        return Err(VAAInvalid.into());
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
        v.write_all(&vaa.emitter_address)?;
        v.write_u64::<BigEndian>(vaa.sequence)?;
        v.write_u8(vaa.consistency_level)?;
        v.write_all(&vaa.payload)?;
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
