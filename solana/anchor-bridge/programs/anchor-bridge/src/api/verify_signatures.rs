use anchor_lang::{prelude::*, solana_program};

use crate::{accounts, anchor_bridge::Bridge, VerifySig, VerifySigsData, MAX_LEN_GUARDIAN_KEYS};
use byteorder::ByteOrder;
use sha3::Digest;
use std::io::Write;

pub const MIN_SECP_PROGRAM_DATA_LEN: usize = 3;

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
    bridge: &mut Bridge,
    ctx: Context<VerifySig>,
    hash: [u8; 32],
    signers: [i8; MAX_LEN_GUARDIAN_KEYS],
    initial_creation: bool,
) -> ProgramResult {
    let sig_infos: Vec<SigInfo> = signers
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

    // We check this manually because the type-level checks are
    // not available for Instructions yet. See the VerifySig
    // struct for more info.
    let ix_acc = &ctx.accounts.instruction_sysvar;
    if *ix_acc.key != solana_program::sysvar::instructions::id() {
        return Err(ProgramError::Custom(42));
    }

    let current_ix_idx =
        solana_program::sysvar::instructions::load_current_index(&ix_acc.try_borrow_data()?);

    if current_ix_idx == 0 {
        return Err(ProgramError::InvalidInstructionData);
    }

    // Retrieve the previous instruction
    let prev_ix_idx = (current_ix_idx - 1) as u8;
    let prev_ix = solana_program::sysvar::instructions::load_instruction_at(
        prev_ix_idx as usize,
        &ix_acc.try_borrow_mut_data()?,
    )
    .map_err(|_e| ProgramError::InvalidAccountData)?;

    // Does prev_ix call the right program?
    if prev_ix.program_id != solana_program::secp256k1_program::id() {
        return Err(ProgramError::InvalidArgument);
    }

    // Is the data correctly sized?
    let prev_data_len = prev_ix.data.len();
    if prev_data_len < MIN_SECP_PROGRAM_DATA_LEN {
        return Err(ProgramError::InvalidAccountData);
    }

    // Parse the instruction data for verification
    let sig_len = prev_ix.data[0];
    let mut index = 1;

    let mut secp_ixs: Vec<SecpInstructionPart> = Vec::with_capacity(sig_len as usize);
    for i in 0..sig_len {
        let sig_offset = byteorder::LE::read_u16(&prev_ix.data[index..index + 2]) as usize;
        index += 2;
        let sig_ix = prev_ix.data[index];
        index += 1;
        let address_offset = byteorder::LE::read_u16(&prev_ix.data[index..index + 2]) as usize;
        index += 2;
        let address_ix = prev_ix.data[index];
        index += 1;
        let msg_offset = byteorder::LE::read_u16(&prev_ix.data[index..index + 2]);
        index += 2;
        let msg_size = byteorder::LE::read_u16(&prev_ix.data[index..index + 2]);
        index += 2;
        let msg_ix = prev_ix.data[index];
        index += 1;

        if address_ix != prev_ix_idx || msg_ix != prev_ix_idx || sig_ix != prev_ix_idx {
            return Err(ProgramError::InvalidArgument);
        }

        let address: &[u8] = &prev_ix.data[address_offset..address_offset + 20];
        let signature: &[u8] = &prev_ix.data[sig_offset..sig_offset + 65];

        // Make sure that all messages are equal
        if i > 0 {
            if msg_offset != secp_ixs[0].msg_offset || msg_size != secp_ixs[0].msg_size {
                return Err(ProgramError::InvalidArgument);
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
        return Err(ProgramError::InvalidArgument);
    }

    // Check message
    let message = &prev_ix.data
        [secp_ixs[0].msg_offset as usize..(secp_ixs[0].msg_offset + secp_ixs[0].msg_size) as usize];

    let mut h = sha3::Keccak256::default();
    if let Err(_) = h.write(message) {
        return Err(ProgramError::InvalidArgument);
    };
    let msg_hash: [u8; 32] = h.finalize().into();
    if msg_hash != hash {
        return Err(ProgramError::InvalidArgument);
    }

    // ------ 8>< *SNIP <>8 --------
    // In original bridge program specific bridge state checks follow

    Ok(())
}
