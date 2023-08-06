#![allow(clippy::collapsible_if)]
use solitaire::*;

use crate::{
    error::Error::{
        GuardianSetMismatch,
        InstructionAtWrongIndex,
        InvalidHash,
        InvalidSecpInstruction,
    },
    GuardianSet,
    GuardianSetDerivationData,
    IsSigned::*,
    SignatureSet,
    MAX_LEN_GUARDIAN_KEYS,
};
use byteorder::ByteOrder;
use solana_program::program_error::ProgramError;
use solitaire::{
    processors::seeded::Seeded,
    CreationLamports::Exempt,
};

#[derive(FromAccounts)]
pub struct VerifySignatures<'b> {
    /// Payer for account creation
    pub payer: Mut<Signer<Info<'b>>>,

    /// Guardian set of the signatures
    pub guardian_set: GuardianSet<'b, { AccountState::Initialized }>,

    /// Signature Account
    pub signature_set: Mut<Signer<SignatureSet<'b, { AccountState::MaybeInitialized }>>>,

    /// Instruction reflection account (special sysvar)
    pub instruction_acc: Info<'b>,
}

impl From<&VerifySignatures<'_>> for GuardianSetDerivationData {
    fn from(data: &VerifySignatures<'_>) -> Self {
        GuardianSetDerivationData {
            index: data.guardian_set.index,
        }
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct VerifySignaturesData {
    /// instruction indices of signers (-1 for missing)
    pub signers: [i8; MAX_LEN_GUARDIAN_KEYS],
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

            Some(SigInfo {
                sig_index: *p as u8,
                signer_index: i as u8,
            })
        })
        .collect();

    let current_instruction =
        solana_program::sysvar::instructions::load_current_index_checked(&accs.instruction_acc)?;
    if current_instruction == 0 {
        return Err(InstructionAtWrongIndex.into());
    }

    // The previous ix must be a secp verification instruction
    let secp_ix_index = (current_instruction - 1) as u8;
    let secp_ix = solana_program::sysvar::instructions::load_instruction_at_checked(
        secp_ix_index as usize,
        &accs.instruction_acc,
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
        let _sig_offset = byteorder::LE::read_u16(&secp_ix.data[index..index + 2]) as usize;
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

        // Make sure that all messages are equal
        if i > 0 {
            if msg_offset != secp_ixs[0].msg_offset || msg_size != secp_ixs[0].msg_size {
                return Err(InvalidSecpInstruction.into());
            }
        }
        secp_ixs.push(SecpInstructionPart {
            address,
            msg_offset,
            msg_size,
        });
    }

    if sig_infos.len() != secp_ixs.len() {
        return Err(ProgramError::InvalidArgument.into());
    }

    // Data must be a hash
    if secp_ixs[0].msg_size != 32 {
        return Err(ProgramError::InvalidArgument.into());
    }

    // Extract message which is encoded in Solana Secp256k1 instruction data.
    let message = &secp_ix.data
        [secp_ixs[0].msg_offset as usize..(secp_ixs[0].msg_offset + secp_ixs[0].msg_size) as usize];

    // Hash the message part, which contains the serialized VAA body.
    let mut msg_hash: [u8; 32] = [0u8; 32];
    msg_hash.copy_from_slice(message);

    if !accs.signature_set.is_initialized() {
        accs.signature_set.signatures = vec![false; accs.guardian_set.keys.len()];
        accs.signature_set.guardian_set_index = accs.guardian_set.index;
        accs.signature_set.hash = msg_hash;

        let size = accs.signature_set.size();
        create_account(
            ctx,
            accs.signature_set.info(),
            accs.payer.key,
            Exempt,
            size,
            ctx.program_id,
            NotSigned,
        )?;
    } else {
        // If the account already existed, check that the parameters match
        if accs.signature_set.guardian_set_index != accs.guardian_set.index {
            return Err(GuardianSetMismatch.into());
        }

        if accs.signature_set.hash != msg_hash {
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
        accs.signature_set.signatures[s.signer_index as usize] = true;
    }

    Ok(())
}
