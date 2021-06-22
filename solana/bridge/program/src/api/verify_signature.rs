use solitaire::*;

use solana_program::{self,};

use crate::{
    accounts::{
        GuardianSet,
        GuardianSetDerivationData,
        SignatureSet,
        SignaturesSetDerivationData,
    },
    types::{self,},
    Error::{
        GuardianSetMismatch,
        InstructionAtWrongIndex,
        InvalidHash,
        InvalidSecpInstruction,
    },
    MAX_LEN_GUARDIAN_KEYS,
};
use byteorder::ByteOrder;
use sha3::Digest;
use solana_program::program_error::ProgramError;
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};
use std::io::Write;
use solana_program::msg;

#[derive(FromAccounts)]
pub struct VerifySignatures<'b> {
    /// Payer for account creation
    pub payer: Signer<Info<'b>>,

    /// Guardian set of the signatures
    pub guardian_set: GuardianSet<'b, { AccountState::Initialized }>,

    /// Signature Account
    pub signature_set: SignatureSet<'b, { AccountState::MaybeInitialized }>,

    /// Instruction reflection account (special sysvar)
    pub instruction_acc: Info<'b>,
}

impl<'b> InstructionContext<'b> for VerifySignatures<'b> {
}

impl From<&VerifySignatures<'_>> for GuardianSetDerivationData {
    fn from(data: &VerifySignatures<'_>) -> Self {
        GuardianSetDerivationData {
            index: data.guardian_set.index,
        }
    }
}

impl From<&VerifySignatures<'_>> for SignaturesSetDerivationData {
    fn from(data: &VerifySignatures<'_>) -> Self {
        SignaturesSetDerivationData {
            // TODO
            hash: data.signature_set.hash,
        }
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct VerifySignaturesData {
    /// Guardian set of the signatures
    pub hash: [u8; 32],
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    /// indicates whether this verification should only succeed if the sig account does not exist
    pub initial_creation: bool,
}

/// SigInfo contains metadata about signers in a VerifySignature ix
struct SigInfo {
    /// index of the signer in the guardianset
    signer_index: u8,
    /// index of the signature in the secp instruction
    sig_index: u8,
}

struct SecpInstructionPart<'a> {
    address: &'a [u8],
    signature: &'a [u8],
    msg_offset: u16,
    msg_size: u16,
}

pub fn verify_signatures(
    ctx: &ExecutionContext,
    accs: &mut VerifySignatures,
    data: VerifySignaturesData,
) -> Result<()> {

    accs.guardian_set
        .verify_derivation(ctx.program_id, &(&*accs).into())?;

    let sig_infos: Vec<SigInfo> = data
        .signers
        .iter()
        .enumerate()
        .filter_map(|(i, p)| {
            if *p == -1 {
                return None;
            }

            return Some(SigInfo {
                sig_index: *p as u8,
                signer_index: i as u8,
            });
        })
        .collect();

    let current_instruction = solana_program::sysvar::instructions::load_current_index(
        &accs.instruction_acc.try_borrow_mut_data()?,
    );
    if current_instruction == 0 {
        return Err(InstructionAtWrongIndex.into());
    }

    // The previous ix must be a secp verification instruction
    let secp_ix_index = (current_instruction - 1) as u8;
    let secp_ix = solana_program::sysvar::instructions::load_instruction_at(
        secp_ix_index as usize,
        &accs.instruction_acc.try_borrow_mut_data()?,
    )
    .map_err(|_| ProgramError::InvalidAccountData)?;

    // Check that the instruction is actually for the secp program
    if secp_ix.program_id != solana_program::secp256k1_program::id() {
        return Err(InvalidSecpInstruction.into());
    }

    let secp_data_len = secp_ix.data.len();
    if secp_data_len < 2 {
        return Err(InvalidSecpInstruction.into());
    }

    let sig_len = secp_ix.data[0];
    let mut index = 1;

    let mut secp_ixs: Vec<SecpInstructionPart> = Vec::with_capacity(sig_len as usize);
    for i in 0..sig_len {
        let sig_offset = byteorder::LE::read_u16(&secp_ix.data[index..index + 2]) as usize;
        index += 2;
        let sig_ix = secp_ix.data[index];
        index += 1;
        let address_offset = byteorder::LE::read_u16(&secp_ix.data[index..index + 2]) as usize;
        index += 2;
        let address_ix = secp_ix.data[index];
        index += 1;
        let msg_offset = byteorder::LE::read_u16(&secp_ix.data[index..index + 2]);
        index += 2;
        let msg_size = byteorder::LE::read_u16(&secp_ix.data[index..index + 2]);
        index += 2;
        let msg_ix = secp_ix.data[index];
        index += 1;

        if address_ix != secp_ix_index || msg_ix != secp_ix_index || sig_ix != secp_ix_index {
            return Err(InvalidSecpInstruction.into());
        }

        let address: &[u8] = &secp_ix.data[address_offset..address_offset + 20];
        let signature: &[u8] = &secp_ix.data[sig_offset..sig_offset + 65];

        // Make sure that all messages are equal
        if i > 0 {
            if msg_offset != secp_ixs[0].msg_offset || msg_size != secp_ixs[0].msg_size {
                return Err(InvalidSecpInstruction.into());
            }
        }
        secp_ixs.push(SecpInstructionPart {
            address,
            signature,
            msg_offset,
            msg_size,
        });
    }

    if sig_infos.len() != secp_ixs.len() {
        return Err(ProgramError::InvalidArgument.into());
    }

    // Check message
    let message = &secp_ix.data
        [secp_ixs[0].msg_offset as usize..(secp_ixs[0].msg_offset + secp_ixs[0].msg_size) as usize];

    let mut h = sha3::Keccak256::default();
    if let Err(e) = h.write(message) {
        return Err(e.into());
    };

    let msg_hash: [u8; 32] = h.finalize().into();
    if msg_hash != data.hash {
        return Err(InvalidHash.into());
    }

    // Track whether the account needs initialization
    // Prepare message/payload-specific sig_info account
    if !accs.signature_set.is_initialized() {
        accs.signature_set.guardian_set_index = accs.guardian_set.index;
        accs.signature_set.hash = data.hash;

        accs.signature_set
            .verify_derivation(ctx.program_id, &(&*accs).into())?;

        accs.signature_set.create(&(&*accs).into(), ctx, accs.payer.key, Exempt)?;
    } else {
        accs.signature_set
            .verify_derivation(ctx.program_id, &(&*accs).into())?;

        // If the account already existed, check that the parameters match
        if accs.signature_set.guardian_set_index != accs.guardian_set.index {
            return Err(GuardianSetMismatch.into());
        }
        if accs.signature_set.hash != data.hash {
            return Err(InvalidHash.into());
        }
    }

    // Write sigs of checked addresses into sig_state
    for s in sig_infos {
        if s.signer_index > accs.guardian_set.num_guardians() {
            return Err(ProgramError::InvalidArgument.into());
        }

        if s.sig_index + 1 > sig_len {
            return Err(ProgramError::InvalidArgument.into());
        }

        let key = accs.guardian_set.keys[s.signer_index as usize];
        // Check key in ix
        if key != secp_ixs[s.sig_index as usize].address {
            return Err(ProgramError::InvalidArgument.into());
        }

        // Overwritten content should be zeros except double signs by the signer or harmless replays
        accs.signature_set.signatures[s.signer_index as usize].0
            .copy_from_slice(&secp_ixs[s.sig_index as usize].signature[0..32]);
        accs.signature_set.signatures[s.signer_index as usize].1
            .copy_from_slice(&secp_ixs[s.sig_index as usize].signature[32..64]);
        accs.signature_set.signatures[s.signer_index as usize].2 = 
            secp_ixs[s.sig_index as usize].signature[64];
    }

    Ok(())
}
