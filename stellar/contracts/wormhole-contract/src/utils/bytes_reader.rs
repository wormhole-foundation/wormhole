use soroban_sdk::{Bytes, BytesN};
use wormhole_interface::WormholeError;

/// A cursor-based reader for parsing bytes in big-endian format.
/// Tracks position and provides clean error handling for out-of-bounds reads.
pub(crate) struct BytesReader<'a> {
    bytes: &'a Bytes,
    cursor: u32,
}

impl<'a> BytesReader<'a> {
    pub(crate) fn new(bytes: &'a Bytes) -> Self {
        Self { bytes, cursor: 0 }
    }

    pub(crate) fn remaining(&self) -> u32 {
        self.bytes.len().saturating_sub(self.cursor)
    }

    fn require(&self, n: u32) -> Result<(), WormholeError> {
        if self.remaining() < n {
            Err(WormholeError::InvalidVAAFormat)
        } else {
            Ok(())
        }
    }

    /// Reads a byte at offset from cursor without bouns checking.
    /// Caller must ensure bounds via `require()` first.
    fn get_unchecked(&self, offset: u32) -> u8 {
        self.bytes.get(self.cursor + offset).unwrap()
    }

    pub(crate) fn read_u8(&mut self) -> Result<u8, WormholeError> {
        self.require(1)?;
        let value = self.get_unchecked(0);
        self.cursor += 1;

        Ok(value)
    }

    pub(crate) fn read_u16_be(&mut self) -> Result<u16, WormholeError> {
        self.require(2)?;
        let value = u16::from(self.get_unchecked(0)) << 8 | u16::from(self.get_unchecked(1));
        self.cursor += 2;

        Ok(value)
    }

    pub(crate) fn read_u32_be(&mut self) -> Result<u32, WormholeError> {
        self.require(4)?;
        let value = (0..4).fold(0u32, |acc, i| (acc << 8) | u32::from(self.get_unchecked(i)));
        self.cursor += 4;

        Ok(value)
    }

    pub(crate) fn read_u64_be(&mut self) -> Result<u64, WormholeError> {
        self.require(8)?;
        let value = (0..8).fold(0u64, |acc, i| (acc << 8) | u64::from(self.get_unchecked(i)));
        self.cursor += 8;

        Ok(value)
    }

    pub(crate) fn read_bytes_n<const N: usize>(&mut self) -> Result<BytesN<N>, WormholeError> {
        let n = u32::try_from(N).map_err(|_| WormholeError::InvalidVAAFormat)?;
        self.read_bytes(n)
            .and_then(|bytes| bytes.try_into().map_err(|_| WormholeError::InvalidVAAFormat))
    }

    pub fn read_bytes(&mut self, len: u32) -> Result<Bytes, WormholeError> {
        self.require(len)?;
        let end = self.cursor.saturating_add(len);
        let slice = self.bytes.slice(self.cursor..end);
        self.cursor = end;
        Ok(slice)
    }

    pub(crate) fn skip(&mut self, n: u32) -> Result<(), WormholeError> {
        self.require(n)?;
        self.cursor = self.cursor.saturating_add(n);
        Ok(())
    }

    pub(crate) fn remaining_bytes(&self) -> Bytes {
        self.bytes.slice(self.cursor..)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use soroban_sdk::Env;

    #[test]
    fn test_read_u8() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x12, 0x34, 0x56]);
        let mut reader = BytesReader::new(&bytes);

        assert_eq!(reader.read_u8().unwrap(), 0x12);
        assert_eq!(reader.read_u8().unwrap(), 0x34);
        assert_eq!(reader.read_u8().unwrap(), 0x56);
        assert!(reader.read_u8().is_err());
    }

    #[test]
    fn test_read_u16_be() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x12, 0x34, 0x56, 0x78]);
        let mut reader = BytesReader::new(&bytes);

        assert_eq!(reader.read_u16_be().unwrap(), 0x1234);
        assert_eq!(reader.read_u16_be().unwrap(), 0x5678);
        assert!(reader.read_u16_be().is_err());
    }

    #[test]
    fn test_read_u32_be() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x12, 0x34, 0x56, 0x78, 0xAB, 0xCD]);
        let mut reader = BytesReader::new(&bytes);

        assert_eq!(reader.read_u32_be().unwrap(), 0x12345678);
        assert!(reader.read_u32_be().is_err());
    }

    #[test]
    fn test_read_u64_be() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0]);
        let mut reader = BytesReader::new(&bytes);

        assert_eq!(reader.read_u64_be().unwrap(), 0x123456789ABCDEF0);
        assert!(reader.read_u64_be().is_err());
    }

    #[test]
    fn test_read_bytes_n() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x01, 0x02, 0x03, 0x04]);
        let mut reader = BytesReader::new(&bytes);

        let result: BytesN<3> = reader.read_bytes_n().unwrap();
        assert_eq!(result.to_array(), [0x01, 0x02, 0x03]);

        assert!(reader.read_bytes_n::<2>().is_err());
    }

    #[test]
    fn test_skip() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x01, 0x02, 0x03, 0x04, 0x05]);
        let mut reader = BytesReader::new(&bytes);

        reader.skip(2).unwrap();
        assert_eq!(reader.read_u8().unwrap(), 0x03);

        assert!(reader.skip(10).is_err());
    }

    #[test]
    fn test_remaining() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x01, 0x02, 0x03, 0x04, 0x05]);
        let mut reader = BytesReader::new(&bytes);

        assert_eq!(reader.remaining(), 5);
        reader.read_u8().unwrap();
        assert_eq!(reader.remaining(), 4);
        reader.skip(3).unwrap();
        assert_eq!(reader.remaining(), 1);
    }

    #[test]
    fn test_remaining_bytes() {
        let env = Env::default();
        let bytes = Bytes::from_array(&env, &[0x01, 0x02, 0x03, 0x04, 0x05]);
        let mut reader = BytesReader::new(&bytes);

        reader.skip(2).unwrap();
        let remaining = reader.remaining_bytes();

        assert_eq!(remaining.get(0).unwrap(), 0x03);
        assert_eq!(remaining.get(1).unwrap(), 0x04);
        assert_eq!(remaining.get(2).unwrap(), 0x05);
    }
}
