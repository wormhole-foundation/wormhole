use soroban_sdk::{Bytes, BytesN, Env};
use wormhole_interface::Error;

/// A cursor-based reader for parsing bytes in big-endian format.
/// Tracks position and provides clean error handling for out-of-bounds reads.
pub(crate) struct BytesReader<'a> {
    bytes: &'a Bytes,
    cursor: u32,
}

impl<'a> BytesReader<'a> {
    pub(crate) fn new(bytes: &'a Bytes) -> Self {
        BytesReader { bytes, cursor: 0 }
    }

    pub(crate) fn remaining(&self) -> u32 {
        self.bytes.len().saturating_sub(self.cursor)
    }

    pub(crate) fn read_u8(&mut self) -> Result<u8, Error> {
        let value = self.bytes.get(self.cursor).ok_or(Error::InvalidVAAFormat)?;
        self.cursor = self.cursor.saturating_add(1);
        Ok(value)
    }

    pub(crate) fn read_u16_be(&mut self) -> Result<u16, Error> {
        let b1 = u16::from(self.read_u8()?);
        let b2 = u16::from(self.read_u8()?);
        Ok((b1 << 8) | b2)
    }

    pub(crate) fn read_u32_be(&mut self) -> Result<u32, Error> {
        let b1 = u32::from(self.read_u8()?);
        let b2 = u32::from(self.read_u8()?);
        let b3 = u32::from(self.read_u8()?);
        let b4 = u32::from(self.read_u8()?);
        Ok((b1 << 24) | (b2 << 16) | (b3 << 8) | b4)
    }

    pub(crate) fn read_u64_be(&mut self) -> Result<u64, Error> {
        let mut result = 0u64;
        for _ in 0..8 {
            result = (result << 8) | u64::from(self.read_u8()?);
        }
        Ok(result)
    }

    #[allow(clippy::needless_range_loop)]
    pub(crate) fn read_bytes_n<const N: usize>(&mut self, env: &Env) -> Result<BytesN<N>, Error> {
        let mut arr = [0u8; N];
        for i in 0..N {
            arr[i] = self.bytes
                .get(self.cursor.saturating_add(u32::try_from(i).unwrap_or(0)))
                .ok_or(Error::InvalidVAAFormat)?;
        }
        self.cursor = self.cursor.saturating_add(u32::try_from(N).unwrap_or(0));
        Ok(BytesN::from_array(env, &arr))
    }

    pub fn read_bytes(&mut self, len: u32) -> Result<Bytes, Error> {
        if self.remaining() < len {
            return Err(Error::InvalidVAAFormat);
        }
        let end = self.cursor.saturating_add(len);
        let slice = self.bytes.slice(self.cursor..end);
        self.cursor = end;
        Ok(slice)
    }

    pub(crate) fn skip(&mut self, n: u32) -> Result<(), Error> {
        if self.remaining() < n {
            return Err(Error::InvalidVAAFormat);
        }
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

        let result: BytesN<3> = reader.read_bytes_n(&env).unwrap();
        assert_eq!(result.to_array(), [0x01, 0x02, 0x03]);

        assert!(reader.read_bytes_n::<2>(&env).is_err());
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
