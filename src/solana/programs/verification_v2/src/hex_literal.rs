// Taken from https://github.com/RustCrypto/utils/blob/master/hex-literal/src/lib.rs
// and slightly modified.
// The package as published is not compatible with the solana toolchain
// due to outdated cargo version, hence the copypaste here.

// #![no_std]
// #![doc = include_str!("../README.md")]
// #![doc(
//     html_logo_url = "https://raw.githubusercontent.com/RustCrypto/media/6ee8e381/logo.svg",
//     html_favicon_url = "https://raw.githubusercontent.com/RustCrypto/media/6ee8e381/logo.svg"
// )]

const fn next_hex_char(string: &[u8], mut pos: usize) -> Option<(u8, usize)> {
    while pos < string.len() {
        let raw_val = string[pos];
        pos += 1;
        let val = match raw_val {
            b'0'..=b'9' => raw_val - 48,
            b'A'..=b'F' => raw_val - 55,
            b'a'..=b'f' => raw_val - 87,
            b' ' | b'\r' | b'\n' | b'\t' => continue,
            0..=127 => panic!("Encountered invalid ASCII character"),
            _ => panic!("Encountered non-ASCII character"),
        };
        return Some((val, pos));
    }
    None
}

const fn next_byte(string: &[u8], pos: usize) -> Option<(u8, usize)> {
    let (half1, pos) = match next_hex_char(string, pos) {
        Some(v) => v,
        None => return None,
    };
    let (half2, pos) = match next_hex_char(string, pos) {
        Some(v) => v,
        None => panic!("Odd number of hex characters"),
    };
    Some(((half1 << 4) + half2, pos))
}

/// Compute length of a byte array which will be decoded from the strings.
///
/// This function is an implementation detail and SHOULD NOT be called directly!
#[doc(hidden)]
pub const fn len(strings: &[&[u8]]) -> usize {
    let mut i = 0;
    let mut len = 0;
    while i < strings.len() {
        let mut pos = 0;
        while let Some((_, new_pos)) = next_byte(strings[i], pos) {
            len += 1;
            pos = new_pos;
        }
        i += 1;
    }
    len
}

/// Decode hex strings into a byte array of pre-computed length.
///
/// This function is an implementation detail and SHOULD NOT be called directly!
#[doc(hidden)]
pub const fn decode<const LEN: usize>(strings: &[&[u8]]) -> Option<[u8; LEN]> {
    let mut string_pos = 0;
    let mut buf = [0u8; LEN];
    let mut buf_pos = 0;
    while string_pos < strings.len() {
        let mut pos = 0;
        let string = &strings[string_pos];
        string_pos += 1;

        while let Some((byte, new_pos)) = next_byte(string, pos) {
            buf[buf_pos] = byte;
            buf_pos += 1;
            pos = new_pos;
        }
    }
    if LEN == buf_pos { Some(buf) } else { None }
}

/// Converts a sequence of hexadecimal string literals to a byte array at compile time.
///
/// See the crate-level docs for more information.
// #[macro_export]
macro_rules! hex {
    ($($s:literal)*) => {{
        const STRINGS: &[&'static [u8]] = &[$($s.as_bytes(),)*];
        const {
            $crate::hex_literal::decode::<{ $crate::hex_literal::len(STRINGS) }>(STRINGS)
                .expect("Output array length should be correct")
        }
    }};
}

pub(crate) use hex;