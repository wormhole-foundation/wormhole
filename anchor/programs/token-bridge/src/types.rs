use std::fmt;

use wormhole_solana_common::utils;
use wormhole_vaas::{Readable, Writeable};

// #[derive(Debug, Clone, PartialEq, Eq)]
// pub struct EncodedString(pub FixedBytes<32>);

// impl From<String> for EncodedString {
//     fn from(value: String) -> Self {
//         let mut bytes = FixedBytes::<32>::default();
//         if value.len() > 32 {
//             bytes.copy_from_slice(&value.as_bytes()[..32]);
//         } else {
//             bytes[..value.len()].copy_from_slice(value.as_bytes());
//         }
//         Self(bytes)
//     }
// }

// impl From<EncodedString> for String {
//     fn from(value: EncodedString) -> Self {
//         let bytes = value.0.trim_end_with(|c| c == '\0');

//         let check_str = String::from_utf8_lossy(bytes).into_owned();
//         println!(
//             "um... {}, {:?}",
//             check_str.as_bytes().len(),
//             check_str.as_bytes()
//         );
//         check_str
//     }
// }

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

    //     #[test]
    //     fn unicode_truncation() {
    //         let pairs = [
    //             // Empty string should not error or mutate.
    //             ("", ""),
    //             // Unicode < 32 should not be corrupted.
    //             ("🔥", "🔥"),
    //             // Unicode @ 32 should not be corrupted.
    //             ("🔥🔥🔥🔥🔥🔥🔥🔥", "🔥🔥🔥🔥🔥🔥🔥🔥"),
    //             // Unicode > 32 should be truncated correctly.
    //             ("🔥🔥🔥🔥🔥🔥🔥🔥🔥", "🔥🔥🔥🔥🔥🔥🔥🔥"),
    //             // Partially overflowing Unicode > 32 should be removed.
    //             // Note: Expecting 31 bytes.
    //             (
    //                 "0000000000000000000000000000000🔥",
    //                 "0000000000000000000000000000000",
    //             ),
    //         ];

    //         for (input, expected) in pairs {
    //             let converted = FixedString::<32>::from(input.to_string());
    //             let recovered = String::from(converted);
    //             assert_eq!(expected, recovered);
    //         }
    //     }
}
