use crate::state::{
    Data,
    Key,
    Metadata,
    EDITION,
    EDITION_MARKER_BIT_SIZE,
    MAX_CREATOR_LIMIT,
    MAX_EDITION_LEN,
    MAX_EDITION_MARKER_SIZE,
    MAX_MASTER_EDITION_LEN,
    MAX_METADATA_LEN,
    MAX_NAME_LENGTH,
    MAX_SYMBOL_LENGTH,
    MAX_URI_LENGTH,
    PREFIX,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::{
    account_info::AccountInfo,
    borsh::try_from_slice_unchecked,
    entrypoint::ProgramResult,
    msg,
    program::{
        invoke,
        invoke_signed,
    },
    program_error::ProgramError,
    program_option::COption,
    program_pack::{
        IsInitialized,
        Pack,
    },
    pubkey::Pubkey,
    system_instruction,
    sysvar::{
        rent::Rent,
        Sysvar,
    },
};
use spl_token::{
    instruction::{
        set_authority,
        AuthorityType,
    },
    state::{
        Account,
        Mint,
    },
};
use std::convert::TryInto;

pub fn try_from_slice_checked<T: BorshDeserialize>(
    data: &[u8],
    data_type: Key,
    data_size: usize,
) -> Option<T> {
    if (data[0] != data_type as u8 && data[0] != Key::Uninitialized as u8)
        || data.len() != data_size
    {
        return None;
    }
    try_from_slice_unchecked(data).ok()
}
