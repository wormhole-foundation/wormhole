#![deny(unused_results)]

pub use chain::*;
pub use error::*;
pub use vaa::*;


pub mod chain;
pub mod vaa;

#[macro_use]
pub mod error;


/// Helper method that attempts to parse and truncate UTF-8 from a byte stream. This is useful when
/// the wire data is expected to contain UTF-8 that is either already truncated, or needs to be,
/// while still maintaining the ability to render.
///
/// This should be used to parse any Text-over-Wormhole fields that are meant to be human readable.
pub(crate) fn parse_fixed_utf8<T: AsRef<[u8]>, const N: usize>(s: T) -> Option<String> {
    use bstr::ByteSlice;
    use std::io::Cursor;
    use std::io::Read;

    // Read Bytes.
    let mut cursor = Cursor::new(s.as_ref());
    let mut buffer = vec![0u8; N];
    cursor.read_exact(&mut buffer).ok()?;
    buffer.retain(|&c| c != 0);

    // Attempt UTF-8 Decoding. Stripping invalid Unicode characters (0xFFFD).
    let mut buffer: Vec<char> = buffer.chars().collect();
    buffer.retain(|&c| c != '\u{FFFD}');

    Some(buffer.iter().collect())
}
