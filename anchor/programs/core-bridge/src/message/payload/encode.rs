use std::io::{Result, Write};

use solana_program::pubkey::Pubkey;

pub trait WormEncode {
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()>;

    #[inline]
    #[doc(hidden)]
    fn u8_slice(slice: &[Self]) -> Option<&[u8]>
    where
        Self: Sized,
    {
        let _ = slice;
        None
    }
}

impl WormEncode for u8 {
    #[inline]
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()> {
        writer.write_all(core::slice::from_ref(self))
    }

    #[inline]
    fn u8_slice(slice: &[Self]) -> Option<&[u8]> {
        Some(slice)
    }
}

/// Integers are encoded as big-endian.
macro_rules! impl_for_integer {
    ($type: ident) => {
        impl WormEncode for $type {
            #[inline]
            fn encode<W: Write>(&self, writer: &mut W) -> Result<()> {
                let bytes = self.to_be_bytes();
                writer.write_all(&bytes)
            }
        }
    };
}

impl_for_integer!(i8);
impl_for_integer!(i16);
impl_for_integer!(i32);
impl_for_integer!(i64);
impl_for_integer!(i128);
impl_for_integer!(u16);
impl_for_integer!(u32);
impl_for_integer!(u64);
impl_for_integer!(u128);

impl WormEncode for bool {
    #[inline]
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()> {
        (u8::from(*self)).encode(writer)
    }
}

impl<T, const N: usize> WormEncode for [T; N]
where
    T: WormEncode,
{
    #[inline]
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()> {
        if N == 0 {
            return Ok(());
        } else {
            for el in self.iter() {
                el.encode(writer)?;
            }
        }
        Ok(())
    }
}

impl WormEncode for Pubkey {
    fn encode<W: Write>(&self, writer: &mut W) -> Result<()> {
        writer.write_all(self.as_ref())
    }
}
