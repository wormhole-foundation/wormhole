use borsh::{io, BorshDeserialize, BorshSerialize};
use solana_program::pubkey::Pubkey;

use super::DataDiscriminator;

/// Wormhole Post Message Shim program message event. This message is encoded
/// as instruction data when the Shim program calls itself via CPI.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, BorshDeserialize, BorshSerialize)]
pub struct MessageEvent {
    pub emitter: Pubkey,
    pub sequence: u64,
    pub submission_time: u32,
}

impl DataDiscriminator for MessageEvent {
    const DISCRIMINATOR: &'static [u8] = &super::MESSAGE_EVENT_DISCRIMINATOR;
}

#[derive(Debug, Clone, PartialEq, Eq, BorshDeserialize, BorshSerialize)]
pub struct GuardianSignatures {
    /// Payer of this guardian signatures account. Only this account may extend
    /// signatures. When the close signatures instruction is invoked, rent will
    /// be returned to this account.
    pub refund_recipient: Pubkey,

    /// Guardian set index that these signatures correspond to. This index will
    /// be checked against the Wormhole Core Bridge program's guardian set when
    /// the verify hash instruction is invoked.
    ///
    /// NOTE: Encoding the guardian set as big-endian matches the derivation
    /// used by the Wormhole Core Bridge program.
    pub guardian_set_index_be: [u8; 4],

    /// Guardian signatures loaded via the post signatures instruction.
    pub guardian_signatures: Vec<[u8; 66]>,
}

impl DataDiscriminator for GuardianSignatures {
    const DISCRIMINATOR: &'static [u8] = &super::GUARDIAN_SIGNATURES_DISCRIMINATOR;
}

/// Deserializes Borsh-serialized data with a discriminator.
pub fn deserialize_with_discriminator<T: BorshDeserialize + DataDiscriminator>(
    data: &[u8],
) -> io::Result<T> {
    if data.len() < T::DISCRIMINATOR.len() || &data[..T::DISCRIMINATOR.len()] != T::DISCRIMINATOR {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "invalid discriminator",
        ));
    }

    T::deserialize(&mut &data[T::DISCRIMINATOR.len()..])
}
