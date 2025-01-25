use solana_program::pubkey::Pubkey;

pub struct GuardianSignatures<'data>(&'data [u8]);

impl<'data> GuardianSignatures<'data> {
    pub const DISCRIMINATOR: [u8; 8] = super::GUARDIAN_SIGNATURES_DISCRIMINATOR;

    pub const MINIMUM_SIZE: usize = {
        8 // discriminator
        + 32 // refund recipient
        + 4 // guardian set index
        + 4 // guardian signatures length
    };

    pub fn new(data: &'data [u8]) -> Option<Self> {
        if data.len() < Self::MINIMUM_SIZE || data[..8] != Self::DISCRIMINATOR {
            return None;
        }

        let account = Self(data);
        let total_len = (account.guardian_signatures_len() as usize)
            .checked_mul(super::GUARDIAN_SIGNATURE_LENGTH)?
            .checked_add(Self::MINIMUM_SIZE)?;

        if data.len() < total_len {
            return None;
        }

        Some(account)
    }

    #[inline]
    pub fn refund_recipient(&self) -> Pubkey {
        Pubkey::new_from_array(self.refund_recipient_slice().try_into().unwrap())
    }

    #[inline]
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_be_bytes(self.guardian_set_index_be())
    }

    #[inline]
    pub fn guardian_set_index_be(&self) -> [u8; 4] {
        self.guardian_index_be_slice().try_into().unwrap()
    }

    #[inline]
    pub fn guardian_signatures_len(&self) -> u32 {
        u32::from_le_bytes(self.guardian_signatures_len_slice().try_into().unwrap())
    }

    #[inline]
    pub fn guardian_signature(
        &self,
        index: usize,
    ) -> Option<[u8; super::GUARDIAN_SIGNATURE_LENGTH]> {
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
            .checked_mul(super::GUARDIAN_SIGNATURE_LENGTH)?
            .checked_add(Self::MINIMUM_SIZE)?;
        let end = start.checked_add(super::GUARDIAN_SIGNATURE_LENGTH)?;
        self.0.get(start..end)
    }
}
