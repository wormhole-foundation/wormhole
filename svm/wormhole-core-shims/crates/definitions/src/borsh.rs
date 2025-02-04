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
    /// signatures. When the close signatures instruction is closed, rent will
    /// be returned to this account.
    pub refund_recipient: Pubkey,

    /// Guardian set index that these signatures correspond to. Storing this
    /// information simplifies the integrator data.
    ///
    /// NOTE: Using big-endian to match the derivation used by the Wormhole
    /// Core Bridge program.
    pub guardian_set_index_be: [u8; 4],

    /// Unverified guardian signatures.
    pub guardian_signatures: Vec<[u8; 66]>,
}

impl DataDiscriminator for GuardianSignatures {
    const DISCRIMINATOR: &'static [u8] = &super::GUARDIAN_SIGNATURES_DISCRIMINATOR;
}

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
