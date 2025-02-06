use crate::GUARDIAN_PUBKEY_LENGTH;

/// Guardian set account owned by the Wormhole Core Bridge program.
///
/// NOTE: This is a zero-copy struct only meant to read in account data. There
/// is no verification of whether this account is owned by the Wormhole Verify
/// VAA program. You must ensure that the account is owned by this program.
///
/// Because this account does not have a discriminator, you must also ensure
/// that the account key matches the PDA's expected key with
/// [find_guardian_set_address].
///
/// [find_guardian_set_address]: crate::find_guardian_set_address
pub struct GuardianSet<'data>(&'data [u8]);

impl<'data> GuardianSet<'data> {
    pub const MINIMUM_SIZE: usize = {
        4 // guardian set index
        + 4 // keys length
        + 4 // creation time
        + 4 // expiration time
    };

    /// Attempts to read a guardian set account from the given data. This method
    /// will return `None` if the data is not a valid guardian set account.
    pub fn new(data: &'data [u8]) -> Option<Self> {
        if data.len() < Self::MINIMUM_SIZE {
            return None;
        }

        let account = Self(data);
        let total_len = (account.keys_len() as usize)
            .checked_mul(GUARDIAN_PUBKEY_LENGTH)?
            .checked_add(Self::MINIMUM_SIZE)?;

        if data.len() < total_len {
            return None;
        }

        Some(account)
    }

    /// Guardian set index that these keys correspond to.
    #[inline]
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.guardian_set_index_slice().try_into().unwrap())
    }

    /// Number of guardian public keys in this set.
    #[inline]
    pub fn keys_len(&self) -> u32 {
        u32::from_le_bytes(self.keys_len_slice().try_into().unwrap())
    }

    /// Guardian public key at the given index. This method will return `None`
    /// if the index is out of bounds.
    #[inline]
    pub fn key(&self, index: usize) -> Option<[u8; GUARDIAN_PUBKEY_LENGTH]> {
        self.key_slice(index)?.try_into().ok()
    }

    /// When the guardian set was created.
    #[inline]
    pub fn creation_time(&self) -> u32 {
        u32::from_le_bytes(self.creation_time_slice().try_into().unwrap())
    }

    /// When the guardian set will expire. A value of 0 means that the guardian
    /// set is the current one.
    #[inline]
    pub fn expiration_time(&self) -> u32 {
        u32::from_le_bytes(self.expiration_time_slice().try_into().unwrap())
    }

    #[inline(always)]
    pub fn guardian_set_index_slice(&self) -> &'data [u8] {
        &self.0[..4]
    }

    #[inline(always)]
    pub fn keys_len_slice(&self) -> &'data [u8] {
        &self.0[4..8]
    }

    #[inline(always)]
    pub fn key_slice(&self, index: usize) -> Option<&'data [u8]> {
        if index >= self.keys_len() as usize {
            return None;
        }

        let start_idx = 8 + index * GUARDIAN_PUBKEY_LENGTH;
        let end_idx = 28 + index * GUARDIAN_PUBKEY_LENGTH;

        Some(&self.0[start_idx..end_idx])
    }

    #[inline(always)]
    pub fn creation_time_slice(&self) -> &'data [u8] {
        let end_idx = self.keys_end_index();
        &self.0[end_idx..(end_idx + 4)]
    }

    #[inline(always)]
    pub fn expiration_time_slice(&self) -> &'data [u8] {
        let end_idx = self.keys_end_index();
        &self.0[(end_idx + 4)..(end_idx + 8)]
    }

    /// Checks if the guardian set is active at the given timestamp. A guardian
    /// set is active if its expiration time is 0 or if the given timestamp is
    /// less than or equal to the expiration time.
    ///
    /// NOTE: This method also handles a special case for the initial guardian
    /// set on mainnet. The initial guardian set was never expired.
    #[inline]
    pub fn is_active(&self, timestamp: u32) -> bool {
        const GUARDIAN_SET_ZERO_INDEX: [u8; 4] = [0, 0, 0, 0];
        const GUARDIAN_SET_ZERO_CREATION_TIME: [u8; 4] = u32::to_le_bytes(1_628_099_186);

        // Note: This is a fix for Wormhole on mainnet.  The initial guardian set was never expired
        // so we block it here.
        if self.guardian_set_index_slice() == GUARDIAN_SET_ZERO_INDEX
            && self.creation_time_slice() == GUARDIAN_SET_ZERO_CREATION_TIME
        {
            false
        } else {
            let expiration_time = self.expiration_time();
            expiration_time == 0 || timestamp <= expiration_time
        }
    }

    /// Returns the number of guardian public keys required to form a quorum.
    #[inline]
    pub fn quorum(&self) -> u32 {
        (self.keys_len() * 2) / 3 + 1
    }

    #[inline(always)]
    fn keys_end_index(&self) -> usize {
        8 + (self.keys_len() as usize) * GUARDIAN_PUBKEY_LENGTH
    }
}

#[cfg(test)]
mod test {
    use base64::{prelude::BASE64_STANDARD, Engine};

    use super::*;

    /// Mainnet guardian set 4: AFEXK4A1BU7BZfi8niAmker98LH9EARB544wKGPXwMyy.
    #[test]
    fn test_guardian_set_active() {
        let encoded_data = "BAAAABMAAABYk7WnbD9zlkVkiIW9zMBs1wo80/9suVJYm96GLCXvQ5ITL7nUpCFXEU3oRgGTvfOi/PgfhqCXZfR2L9EQegCGsy16CXeSaiBRMdhzHTnL64yCsv2C+u0nEdWa8PJJnRbnJvayEbOXVsBCRBvm2GULabVOvnFeI0NUzltNNI+3S5WOiWbi7D29SVinzRXnyvB8Tj3I58Rp+SyM2I+4AFogdKO/kTlT1pUmDYi8GqJaTu42PvAACsAHZyezX76i2sKP7lzLD+p2jq9FztE2udniSQNGSuiJ9cinI/wU+TEkt8c4hDy7iehkyGLDjN3Mz5XSzDek3ANqjSMrSPYs3UcxQS9IkNp5j2iWozMfZLSMEtHVf9nL5wgRcaob4dNsr+OGeRD5nAnjR4mcGcOBkrbnOHzNdoJ3wX2rG3pQJ8CzzxeOIa0ud64GcRVJz7sfnHqdgJboXhSH81UV0CqSdTUEqNdUcbn0nttvvryJj0A+R3PpX+sV6Ayamcg0jXiZHmYAAAAA";

        let data = BASE64_STANDARD.decode(encoded_data).unwrap();

        let guardian_set = GuardianSet::new(&data).unwrap();
        assert_eq!(guardian_set.guardian_set_index(), 4);
        assert_eq!(guardian_set.keys_len(), 19);
        assert_eq!(guardian_set.creation_time(), 1_713_281_400);
        assert_eq!(guardian_set.expiration_time(), 0);
        assert!(guardian_set.is_active(0));
        assert!(guardian_set.is_active(u32::MAX));

        let expected_keys = [
            [
                0x58, 0x93, 0xb5, 0xa7, 0x6c, 0x3f, 0x73, 0x96, 0x45, 0x64, 0x88, 0x85, 0xbd, 0xcc,
                0xc0, 0x6c, 0xd7, 0x0a, 0x3c, 0xd3,
            ],
            [
                0xff, 0x6c, 0xb9, 0x52, 0x58, 0x9b, 0xde, 0x86, 0x2c, 0x25, 0xef, 0x43, 0x92, 0x13,
                0x2f, 0xb9, 0xd4, 0xa4, 0x21, 0x57,
            ],
            [
                0x11, 0x4d, 0xe8, 0x46, 0x01, 0x93, 0xbd, 0xf3, 0xa2, 0xfc, 0xf8, 0x1f, 0x86, 0xa0,
                0x97, 0x65, 0xf4, 0x76, 0x2f, 0xd1,
            ],
            [
                0x10, 0x7a, 0x00, 0x86, 0xb3, 0x2d, 0x7a, 0x09, 0x77, 0x92, 0x6a, 0x20, 0x51, 0x31,
                0xd8, 0x73, 0x1d, 0x39, 0xcb, 0xeb,
            ],
            [
                0x8c, 0x82, 0xb2, 0xfd, 0x82, 0xfa, 0xed, 0x27, 0x11, 0xd5, 0x9a, 0xf0, 0xf2, 0x49,
                0x9d, 0x16, 0xe7, 0x26, 0xf6, 0xb2,
            ],
            [
                0x11, 0xb3, 0x97, 0x56, 0xc0, 0x42, 0x44, 0x1b, 0xe6, 0xd8, 0x65, 0x0b, 0x69, 0xb5,
                0x4e, 0xbe, 0x71, 0x5e, 0x23, 0x43,
            ],
            [
                0x54, 0xce, 0x5b, 0x4d, 0x34, 0x8f, 0xb7, 0x4b, 0x95, 0x8e, 0x89, 0x66, 0xe2, 0xec,
                0x3d, 0xbd, 0x49, 0x58, 0xa7, 0xcd,
            ],
            [
                0x15, 0xe7, 0xca, 0xf0, 0x7c, 0x4e, 0x3d, 0xc8, 0xe7, 0xc4, 0x69, 0xf9, 0x2c, 0x8c,
                0xd8, 0x8f, 0xb8, 0x00, 0x5a, 0x20,
            ],
            [
                0x74, 0xa3, 0xbf, 0x91, 0x39, 0x53, 0xd6, 0x95, 0x26, 0x0d, 0x88, 0xbc, 0x1a, 0xa2,
                0x5a, 0x4e, 0xee, 0x36, 0x3e, 0xf0,
            ],
            [
                0x00, 0x0a, 0xc0, 0x07, 0x67, 0x27, 0xb3, 0x5f, 0xbe, 0xa2, 0xda, 0xc2, 0x8f, 0xee,
                0x5c, 0xcb, 0x0f, 0xea, 0x76, 0x8e,
            ],
            [
                0xaf, 0x45, 0xce, 0xd1, 0x36, 0xb9, 0xd9, 0xe2, 0x49, 0x03, 0x46, 0x4a, 0xe8, 0x89,
                0xf5, 0xc8, 0xa7, 0x23, 0xfc, 0x14,
            ],
            [
                0xf9, 0x31, 0x24, 0xb7, 0xc7, 0x38, 0x84, 0x3c, 0xbb, 0x89, 0xe8, 0x64, 0xc8, 0x62,
                0xc3, 0x8c, 0xdd, 0xcc, 0xcf, 0x95,
            ],
            [
                0xd2, 0xcc, 0x37, 0xa4, 0xdc, 0x03, 0x6a, 0x8d, 0x23, 0x2b, 0x48, 0xf6, 0x2c, 0xdd,
                0x47, 0x31, 0x41, 0x2f, 0x48, 0x90,
            ],
            [
                0xda, 0x79, 0x8f, 0x68, 0x96, 0xa3, 0x33, 0x1f, 0x64, 0xb4, 0x8c, 0x12, 0xd1, 0xd5,
                0x7f, 0xd9, 0xcb, 0xe7, 0x08, 0x11,
            ],
            [
                0x71, 0xaa, 0x1b, 0xe1, 0xd3, 0x6c, 0xaf, 0xe3, 0x86, 0x79, 0x10, 0xf9, 0x9c, 0x09,
                0xe3, 0x47, 0x89, 0x9c, 0x19, 0xc3,
            ],
            [
                0x81, 0x92, 0xb6, 0xe7, 0x38, 0x7c, 0xcd, 0x76, 0x82, 0x77, 0xc1, 0x7d, 0xab, 0x1b,
                0x7a, 0x50, 0x27, 0xc0, 0xb3, 0xcf,
            ],
            [
                0x17, 0x8e, 0x21, 0xad, 0x2e, 0x77, 0xae, 0x06, 0x71, 0x15, 0x49, 0xcf, 0xbb, 0x1f,
                0x9c, 0x7a, 0x9d, 0x80, 0x96, 0xe8,
            ],
            [
                0x5e, 0x14, 0x87, 0xf3, 0x55, 0x15, 0xd0, 0x2a, 0x92, 0x75, 0x35, 0x04, 0xa8, 0xd7,
                0x54, 0x71, 0xb9, 0xf4, 0x9e, 0xdb,
            ],
            [
                0x6f, 0xbe, 0xbc, 0x89, 0x8f, 0x40, 0x3e, 0x47, 0x73, 0xe9, 0x5f, 0xeb, 0x15, 0xe8,
                0x0c, 0x9a, 0x99, 0xc8, 0x34, 0x8d,
            ],
        ];

        for (i, expected_key) in expected_keys.iter().enumerate() {
            let key = guardian_set.key(i).unwrap();
            assert_eq!(&key, expected_key, "key[{}]", i);
        }

        // Add a byte at the end and we should expect the same output.
        let mut data = BASE64_STANDARD.decode(encoded_data).unwrap();
        data.push(0);

        let guardian_set = GuardianSet::new(&data).unwrap();
        assert_eq!(guardian_set.guardian_set_index(), 4);
        assert_eq!(guardian_set.keys_len(), 19);
        assert_eq!(guardian_set.creation_time(), 1_713_281_400);
        assert_eq!(guardian_set.expiration_time(), 0);
        assert!(guardian_set.is_active(0));
        assert!(guardian_set.is_active(u32::MAX));

        for (i, expected_key) in expected_keys.iter().enumerate() {
            let key = guardian_set.key(i).unwrap();
            assert_eq!(&key, expected_key, "key[{}]", i);
        }

        // Remove two bytes from this data and we should expect None.
        data.pop();
        data.pop();

        assert!(GuardianSet::new(&data).is_none());
    }

    /// Mainnet guardian set 0: XMDS7qfSAgYsonPpKoAjcGhX9VFjXdGkiHjEDkTidf8H2P.
    #[test]
    fn test_guardian_set_zero() {
        let encoded_data = "AAAAAAEAAABYzDrlwJeyE848gZeeG5+VcHRqpXLSCmEAAAAA";

        let data = BASE64_STANDARD.decode(encoded_data).unwrap();

        let guardian_set = GuardianSet::new(&data).unwrap();
        assert_eq!(guardian_set.guardian_set_index(), 0);
        assert_eq!(guardian_set.keys_len(), 1);
        assert_eq!(guardian_set.creation_time(), 1_628_099_186);
        assert_eq!(guardian_set.expiration_time(), 0);
        assert!(!guardian_set.is_active(0));
        assert!(!guardian_set.is_active(u32::MAX));

        let expected_keys = [[
            0x58, 0xcc, 0x3a, 0xe5, 0xc0, 0x97, 0xb2, 0x13, 0xce, 0x3c, 0x81, 0x97, 0x9e, 0x1b,
            0x9f, 0x95, 0x70, 0x74, 0x6a, 0xa5,
        ]];

        for (i, expected_key) in expected_keys.iter().enumerate() {
            let key = guardian_set.key(i).unwrap();
            assert_eq!(&key, expected_key, "key[{}]", i);
        }
    }

    /// Mainnet guardian set 3: 6d3w8mGjJauf6gCAg7WfLezbaPmUHYGuoNutnfYF1RYM.
    #[test]
    fn test_guardian_set_expired() {
        let encoded_data = "AwAAABMAAABYzDrlwJeyE848gZeeG5+VcHRqpf9suVJYm96GLCXvQ5ITL7nUpCFXEU3oRgGTvfOi/PgfhqCXZfR2L9EQegCGsy16CXeSaiBRMdhzHTnL64yCsv2C+u0nEdWa8PJJnRbnJvayEbOXVsBCRBvm2GULabVOvnFeI0NUzltNNI+3S5WOiWbi7D29SVinzRXnyvB8Tj3I58Rp+SyM2I+4AFogdKO/kTlT1pUmDYi8GqJaTu42PvAACsAHZyezX76i2sKP7lzLD+p2jq9FztE2udniSQNGSuiJ9cinI/wU+TEkt8c4hDy7iehkyGLDjN3Mz5XSzDek3ANqjSMrSPYs3UcxQS9IkNp5j2iWozMfZLSMEtHVf9nL5wgRcaob4dNsr+OGeRD5nAnjR4mcGcOBkrbnOHzNdoJ3wX2rG3pQJ8CzzxeOIa0ud64GcRVJz7sfnHqdgJboXhSH81UV0CqSdTUEqNdUcbn0nttvvryJj0A+R3PpX+sV6Ayamcg0jUA8xWP46h9m";

        let data = BASE64_STANDARD.decode(encoded_data).unwrap();

        let guardian_set = GuardianSet::new(&data).unwrap();
        assert_eq!(guardian_set.guardian_set_index(), 3);
        assert_eq!(guardian_set.keys_len(), 19);
        assert_eq!(guardian_set.creation_time(), 1_673_870_400);
        assert_eq!(guardian_set.expiration_time(), 1_713_367_800);
        assert!(guardian_set.is_active(1_713_367_800));
        assert!(!guardian_set.is_active(1_713_367_801));

        let expected_keys = [
            [
                0x58, 0xcc, 0x3a, 0xe5, 0xc0, 0x97, 0xb2, 0x13, 0xce, 0x3c, 0x81, 0x97, 0x9e, 0x1b,
                0x9f, 0x95, 0x70, 0x74, 0x6a, 0xa5,
            ],
            [
                0xff, 0x6c, 0xb9, 0x52, 0x58, 0x9b, 0xde, 0x86, 0x2c, 0x25, 0xef, 0x43, 0x92, 0x13,
                0x2f, 0xb9, 0xd4, 0xa4, 0x21, 0x57,
            ],
            [
                0x11, 0x4d, 0xe8, 0x46, 0x01, 0x93, 0xbd, 0xf3, 0xa2, 0xfc, 0xf8, 0x1f, 0x86, 0xa0,
                0x97, 0x65, 0xf4, 0x76, 0x2f, 0xd1,
            ],
            [
                0x10, 0x7a, 0x00, 0x86, 0xb3, 0x2d, 0x7a, 0x09, 0x77, 0x92, 0x6a, 0x20, 0x51, 0x31,
                0xd8, 0x73, 0x1d, 0x39, 0xcb, 0xeb,
            ],
            [
                0x8c, 0x82, 0xb2, 0xfd, 0x82, 0xfa, 0xed, 0x27, 0x11, 0xd5, 0x9a, 0xf0, 0xf2, 0x49,
                0x9d, 0x16, 0xe7, 0x26, 0xf6, 0xb2,
            ],
            [
                0x11, 0xb3, 0x97, 0x56, 0xc0, 0x42, 0x44, 0x1b, 0xe6, 0xd8, 0x65, 0x0b, 0x69, 0xb5,
                0x4e, 0xbe, 0x71, 0x5e, 0x23, 0x43,
            ],
            [
                0x54, 0xce, 0x5b, 0x4d, 0x34, 0x8f, 0xb7, 0x4b, 0x95, 0x8e, 0x89, 0x66, 0xe2, 0xec,
                0x3d, 0xbd, 0x49, 0x58, 0xa7, 0xcd,
            ],
            [
                0x15, 0xe7, 0xca, 0xf0, 0x7c, 0x4e, 0x3d, 0xc8, 0xe7, 0xc4, 0x69, 0xf9, 0x2c, 0x8c,
                0xd8, 0x8f, 0xb8, 0x00, 0x5a, 0x20,
            ],
            [
                0x74, 0xa3, 0xbf, 0x91, 0x39, 0x53, 0xd6, 0x95, 0x26, 0x0d, 0x88, 0xbc, 0x1a, 0xa2,
                0x5a, 0x4e, 0xee, 0x36, 0x3e, 0xf0,
            ],
            [
                0x00, 0x0a, 0xc0, 0x07, 0x67, 0x27, 0xb3, 0x5f, 0xbe, 0xa2, 0xda, 0xc2, 0x8f, 0xee,
                0x5c, 0xcb, 0x0f, 0xea, 0x76, 0x8e,
            ],
            [
                0xaf, 0x45, 0xce, 0xd1, 0x36, 0xb9, 0xd9, 0xe2, 0x49, 0x03, 0x46, 0x4a, 0xe8, 0x89,
                0xf5, 0xc8, 0xa7, 0x23, 0xfc, 0x14,
            ],
            [
                0xf9, 0x31, 0x24, 0xb7, 0xc7, 0x38, 0x84, 0x3c, 0xbb, 0x89, 0xe8, 0x64, 0xc8, 0x62,
                0xc3, 0x8c, 0xdd, 0xcc, 0xcf, 0x95,
            ],
            [
                0xd2, 0xcc, 0x37, 0xa4, 0xdc, 0x03, 0x6a, 0x8d, 0x23, 0x2b, 0x48, 0xf6, 0x2c, 0xdd,
                0x47, 0x31, 0x41, 0x2f, 0x48, 0x90,
            ],
            [
                0xda, 0x79, 0x8f, 0x68, 0x96, 0xa3, 0x33, 0x1f, 0x64, 0xb4, 0x8c, 0x12, 0xd1, 0xd5,
                0x7f, 0xd9, 0xcb, 0xe7, 0x08, 0x11,
            ],
            [
                0x71, 0xaa, 0x1b, 0xe1, 0xd3, 0x6c, 0xaf, 0xe3, 0x86, 0x79, 0x10, 0xf9, 0x9c, 0x09,
                0xe3, 0x47, 0x89, 0x9c, 0x19, 0xc3,
            ],
            [
                0x81, 0x92, 0xb6, 0xe7, 0x38, 0x7c, 0xcd, 0x76, 0x82, 0x77, 0xc1, 0x7d, 0xab, 0x1b,
                0x7a, 0x50, 0x27, 0xc0, 0xb3, 0xcf,
            ],
            [
                0x17, 0x8e, 0x21, 0xad, 0x2e, 0x77, 0xae, 0x06, 0x71, 0x15, 0x49, 0xcf, 0xbb, 0x1f,
                0x9c, 0x7a, 0x9d, 0x80, 0x96, 0xe8,
            ],
            [
                0x5e, 0x14, 0x87, 0xf3, 0x55, 0x15, 0xd0, 0x2a, 0x92, 0x75, 0x35, 0x04, 0xa8, 0xd7,
                0x54, 0x71, 0xb9, 0xf4, 0x9e, 0xdb,
            ],
            [
                0x6f, 0xbe, 0xbc, 0x89, 0x8f, 0x40, 0x3e, 0x47, 0x73, 0xe9, 0x5f, 0xeb, 0x15, 0xe8,
                0x0c, 0x9a, 0x99, 0xc8, 0x34, 0x8d,
            ],
        ];

        for (i, expected_key) in expected_keys.iter().enumerate() {
            let key = guardian_set.key(i).unwrap();
            assert_eq!(&key, expected_key, "key[{}]", i);
        }
    }
}
