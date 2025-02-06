use solana_program::pubkey::Pubkey;

use crate::{GUARDIAN_SIGNATURES_DISCRIMINATOR, GUARDIAN_SIGNATURE_LENGTH};

/// Guardian signatures account owned by the Wormhole Verify VAA program.
///
/// NOTE: This is a zero-copy struct only meant to read in account data. There
/// is no verification of whether this account is owned by the Wormhole Verify
/// VAA program. You must ensure that the account is owned by this program.
pub struct GuardianSignatures<'data>(&'data [u8]);

impl<'data> GuardianSignatures<'data> {
    pub const DISCRIMINATOR: [u8; 8] = GUARDIAN_SIGNATURES_DISCRIMINATOR;

    pub const MINIMUM_SIZE: usize = {
        8 // discriminator
        + 32 // refund recipient
        + 4 // guardian set index
        + 4 // guardian signatures length
    };

    /// Attempts to read a guardian signatures account from the given data. This
    /// method will return `None` if the data is not a valid guardian signatures
    /// account.
    pub fn new(data: &'data [u8]) -> Option<Self> {
        if data.len() < Self::MINIMUM_SIZE || data[..8] != Self::DISCRIMINATOR {
            return None;
        }

        let account = Self(data);
        let total_len = (account.guardian_signatures_len() as usize)
            .checked_mul(GUARDIAN_SIGNATURE_LENGTH)?
            .checked_add(Self::MINIMUM_SIZE)?;

        if data.len() < total_len {
            return None;
        }

        Some(account)
    }

    /// Payer of this guardian signatures account. Only this account may extend
    /// signatures. When the close signatures instruction is invoked, rent will
    /// be returned to this account.
    #[inline]
    pub fn refund_recipient(&self) -> Pubkey {
        Pubkey::new_from_array(self.refund_recipient_slice().try_into().unwrap())
    }

    /// Guardian set index that these signatures correspond to. This index will
    /// be checked against the Wormhole Core Bridge program's guardian set when
    /// the verify hash instruction is invoked.
    #[inline]
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_be_bytes(self.guardian_set_index_be())
    }

    /// Guardian set index that these signatures correspond to. This index will
    /// be checked against the Wormhole Core Bridge program's guardian set when
    /// the verify hash instruction is invoked.
    ///
    /// NOTE: Encoding the guardian set as big-endian matches the derivation
    /// used by the Wormhole Core Bridge program.
    #[inline]
    pub fn guardian_set_index_be(&self) -> [u8; 4] {
        self.guardian_index_be_slice().try_into().unwrap()
    }

    /// Number of guardian signatures in this account.
    #[inline]
    pub fn guardian_signatures_len(&self) -> u32 {
        u32::from_le_bytes(self.guardian_signatures_len_slice().try_into().unwrap())
    }

    /// Guardian signature at the given index. This method will return `None` if
    /// the index is out of bounds.
    #[inline]
    pub fn guardian_signature(&self, index: usize) -> Option<[u8; GUARDIAN_SIGNATURE_LENGTH]> {
        let signature = self.guardian_signature_slice(index)?;
        Some(signature.try_into().unwrap())
    }

    #[inline(always)]
    pub fn refund_recipient_slice(&self) -> &'data [u8] {
        &self.0[8..40]
    }

    #[inline(always)]
    pub fn guardian_index_be_slice(&self) -> &'data [u8] {
        &self.0[40..44]
    }

    #[inline(always)]
    pub fn guardian_signatures_len_slice(&self) -> &'data [u8] {
        &self.0[44..48]
    }

    #[inline(always)]
    pub fn guardian_signature_slice(&self, index: usize) -> Option<&'data [u8]> {
        let start = index
            .checked_mul(GUARDIAN_SIGNATURE_LENGTH)?
            .checked_add(Self::MINIMUM_SIZE)?;
        let end = start.checked_add(GUARDIAN_SIGNATURE_LENGTH)?;
        self.0.get(start..end)
    }
}
