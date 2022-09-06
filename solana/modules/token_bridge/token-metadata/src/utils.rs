use crate::state::Key;
use borsh::BorshDeserialize;
use solana_program::borsh::try_from_slice_unchecked;

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
