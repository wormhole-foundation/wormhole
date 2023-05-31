use std::{
    fmt, io,
    ops::{Deref, DerefMut},
};

use core_bridge_program::{WormDecode, WormEncode};
use wormhole_common::utils;

use crate::constants::MAX_DECIMALS;

#[derive(Debug, Copy, Clone, PartialEq, Eq, PartialOrd, Ord)]
pub struct NormalizedAmount {
    value: u64,
}

impl NormalizedAmount {
    pub fn from_raw(value: u64, decimals: u8) -> Self {
        if decimals <= MAX_DECIMALS {
            NormalizedAmount { value }
        } else {
            NormalizedAmount {
                value: value / u64::pow(10, u32::from(decimals - MAX_DECIMALS)),
            }
        }
    }

    pub fn checked_to_raw(self, decimals: u8) -> Option<u64> {
        if decimals <= MAX_DECIMALS {
            Some(self.value)
        } else {
            self.value
                .checked_mul(u64::pow(10, u32::from(decimals - MAX_DECIMALS)))
        }
    }
}

impl From<u64> for NormalizedAmount {
    fn from(value: u64) -> Self {
        NormalizedAmount { value }
    }
}

impl From<NormalizedAmount> for u64 {
    fn from(amount: NormalizedAmount) -> Self {
        amount.value
    }
}

impl WormDecode for NormalizedAmount {
    fn decode_reader<R: std::io::Read>(reader: &mut R) -> std::io::Result<Self> {
        let mut zeros = [0; 24];
        reader.read_exact(&mut zeros)?;

        if utils::is_nonzero_array(&zeros) {
            return Err(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                "u64 overflow",
            ));
        }

        let value = u64::decode_reader(reader)?;
        Ok(NormalizedAmount { value })
    }
}

impl WormEncode for NormalizedAmount {
    fn encode<W: std::io::Write>(&self, writer: &mut W) -> std::io::Result<()> {
        [0u8; 24].encode(writer)?;
        self.value.encode(writer)
    }
}

impl fmt::Display for NormalizedAmount {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FixedString<const N: usize>([u8; N]);

impl<const N: usize> Deref for FixedString<N> {
    type Target = [u8; N];

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl<const N: usize> DerefMut for FixedString<N> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl<const N: usize> AsRef<[u8]> for FixedString<N> {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

impl<const N: usize> From<String> for FixedString<N> {
    fn from(value: String) -> Self {
        let mut bytes = [0; N];
        if value.len() > N {
            bytes.copy_from_slice(&value.as_bytes()[..N]);
        } else {
            bytes[..value.len()].copy_from_slice(value.as_bytes());
        }

        Self(bytes)
    }
}

impl<const N: usize> From<FixedString<N>> for String {
    fn from(value: FixedString<N>) -> Self {
        if N == 0 {
            String::new()
        } else {
            // Behold, craziness.
            use bstr::ByteSlice;
            value
                .chars()
                .filter(|&c| c != '\u{FFFD}')
                .collect::<Vec<char>>()
                .iter()
                .collect::<String>()
                .trim_end_matches(char::from(0))
                .to_string()
        }
    }
}

impl<const N: usize> WormDecode for FixedString<N> {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let mut bytes = [0; N];
        reader.read_exact(&mut bytes)?;
        Ok(Self(bytes))
    }
}

impl<const N: usize> WormEncode for FixedString<N> {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        writer.write_all(&self.0)
    }
}

#[cfg(test)]
mod test {
    use super::*;

    const TRIAL_AMOUNTS: [u64; 3] = [0, 69420, u64::MAX];

    #[test]
    fn normalized_amount_7_decimals() {
        const DECIMALS: u8 = 7;

        for &amount in TRIAL_AMOUNTS.iter() {
            let normalized = NormalizedAmount::from_raw(amount, DECIMALS);
            assert_eq!(u64::from(normalized), amount);
            let recovered = NormalizedAmount::checked_to_raw(normalized, DECIMALS).unwrap();
            assert_eq!(recovered, amount);
        }
    }

    #[test]
    fn normalized_amount_8_decimals() {
        const DECIMALS: u8 = 8;

        for &amount in TRIAL_AMOUNTS.iter() {
            let normalized = NormalizedAmount::from_raw(amount, DECIMALS);
            assert_eq!(u64::from(normalized), amount);
            let recovered = NormalizedAmount::checked_to_raw(normalized, DECIMALS).unwrap();
            assert_eq!(recovered, amount);
        }
    }

    #[test]
    fn normalized_amount_9_decimals() {
        const DECIMALS: u8 = 9;

        for &amount in TRIAL_AMOUNTS.iter() {
            let normalized = NormalizedAmount::from_raw(amount, DECIMALS);
            assert_eq!(u64::from(normalized), amount / 10);

            // Recovered amount will be truncated.
            let recovered = NormalizedAmount::checked_to_raw(normalized, DECIMALS).unwrap();
            assert_eq!(recovered, 10 * (amount / 10));
        }
    }

    #[test]
    fn normalized_amount_too_large() {
        let recovered = NormalizedAmount::checked_to_raw(u64::MAX.into(), 9);
        assert_eq!(recovered, None);
    }

    #[test]
    fn unicode_truncation() {
        let pairs = [
            // Empty string should not error or mutate.
            ("", ""),
            // Unicode < 32 should not be corrupted.
            ("🔥", "🔥"),
            // Unicode @ 32 should not be corrupted.
            ("🔥🔥🔥🔥🔥🔥🔥🔥", "🔥🔥🔥🔥🔥🔥🔥🔥"),
            // Unicode > 32 should be truncated correctly.
            ("🔥🔥🔥🔥🔥🔥🔥🔥🔥", "🔥🔥🔥🔥🔥🔥🔥🔥"),
            // Partially overflowing Unicode > 32 should be removed.
            // Note: Expecting 31 bytes.
            (
                "0000000000000000000000000000000🔥",
                "0000000000000000000000000000000",
            ),
        ];

        for (input, expected) in pairs {
            let converted = FixedString::<32>::from(input.to_string());
            let recovered = String::from(converted);
            assert_eq!(expected, recovered);
        }
    }
}
