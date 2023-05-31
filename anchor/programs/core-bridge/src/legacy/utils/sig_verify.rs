use anchor_lang::prelude::*;
use solana_program::{instruction::Instruction, program_error::ProgramError};

const SIGNATURE_LEN: usize = 65;
const ETH_PUBKEY_LEN: usize = 20;

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy)]
pub struct SigVerifyOffsets {
    pub signature_offset: u16, // offset to [signature,recovery_id,etherum_address] of 64+1+20 bytes
    pub signature_ix_index: u8, // instruction index to find data
    pub eth_pubkey_offset: u16, // offset to [signature,recovery_id] of 64+1 bytes
    pub eth_pubkey_ix_index: u8, // instruction index to find data
    pub message_offset: u16,   // offset to start of message data
    pub message_size: u16,     // size of message data
    pub message_ix_index: u8,  // index of instruction data to get message data
}

impl SigVerifyOffsets {
    pub const LEN: usize = 2    // signature_key_offset
        + 1                     // signature_instruction_index
        + 2                     // pubkey_offset
        + 1                     // pubkey_instruction_index
        + 2                     // message_data_offset
        + 2                     // message_data_size
        + 1                     // message_instruction_index
    ;
}

pub(in crate::legacy) struct SigVerifyParameters {
    pub offsets: SigVerifyOffsets,
    pub signature: [u8; SIGNATURE_LEN],
    pub eth_pubkey: [u8; ETH_PUBKEY_LEN],
    pub message: Vec<u8>,
}

pub(in crate::legacy) fn deserialize_secp256k1_ix(
    ix: &Instruction,
) -> Result<Vec<SigVerifyParameters>> {
    // Check that the program invoked is the secp256k1 program.
    require_keys_eq!(ix.program_id, solana_program::secp256k1_program::id());
    require_eq!(ix.accounts.len(), 0);

    let ix_data = &ix.data;

    // First byte encodes the number of signatures.
    let mut params = Vec::with_capacity(ix_data[0].into());

    // For each offset encoded, grab each SigVerify parameter (signature, eth pubkey, message).
    for i in 0..params.capacity() {
        let offsets_idx = 1 + i * SigVerifyOffsets::LEN;
        let offsets = SigVerifyOffsets::deserialize(
            &mut &ix_data[offsets_idx..(offsets_idx + SigVerifyOffsets::LEN)],
        )
        .map_err(|_| ProgramError::InvalidAccountData)?;

        let mut data = SigVerifyParameters {
            offsets,
            signature: [0; SIGNATURE_LEN],
            eth_pubkey: [0; ETH_PUBKEY_LEN],
            message: Vec::with_capacity(offsets.message_size.into()),
        };

        let signature_offset = usize::from(offsets.signature_offset);
        data.signature
            .copy_from_slice(&ix_data[signature_offset..(signature_offset + SIGNATURE_LEN)]);

        let eth_pubkey_offset = usize::from(offsets.eth_pubkey_offset);
        data.eth_pubkey
            .copy_from_slice(&ix_data[eth_pubkey_offset..(eth_pubkey_offset + ETH_PUBKEY_LEN)]);

        let message_offset = usize::from(offsets.message_offset);
        data.message.extend_from_slice(
            &ix_data[message_offset..(message_offset + data.message.capacity())],
        );

        params.push(data);
    }

    Ok(params)
}
