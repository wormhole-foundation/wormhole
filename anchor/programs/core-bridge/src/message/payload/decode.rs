use core::mem::size_of;
use std::io::{Error, ErrorKind, Read, Result};

use solana_program::pubkey::Pubkey;

const ERROR_UNEXPECTED_LENGTH_OF_INPUT: &str = "Unexpected length of input";

pub trait WormDecode: Sized {
    fn decode(buf: &mut &[u8]) -> Result<Self> {
        Self::decode_reader(&mut *buf)
    }

    fn decode_reader<R: Read>(reader: &mut R) -> Result<Self>;

    #[inline]
    #[doc(hidden)]
    fn array_from_reader<R: Read, const N: usize>(reader: &mut R) -> Result<Option<[Self; N]>> {
        let _ = reader;
        Ok(None)
    }
}

impl WormDecode for u8 {
    #[inline]
    fn decode_reader<R: Read>(reader: &mut R) -> Result<Self> {
        let mut buf = [0u8; 1];
        reader
            .read_exact(&mut buf)
            .map_err(unexpected_eof_to_unexpected_length_of_input)?;
        Ok(buf[0])
    }

    #[inline]
    #[doc(hidden)]
    fn array_from_reader<R: Read, const N: usize>(reader: &mut R) -> Result<Option<[Self; N]>> {
        let mut arr = [0u8; N];
        reader
            .read_exact(&mut arr)
            .map_err(unexpected_eof_to_unexpected_length_of_input)?;
        Ok(Some(arr))
    }
}

fn unexpected_eof_to_unexpected_length_of_input(e: Error) -> Error {
    if e.kind() == ErrorKind::UnexpectedEof {
        Error::new(ErrorKind::InvalidInput, ERROR_UNEXPECTED_LENGTH_OF_INPUT)
    } else {
        e
    }
}

/// Integers are encoded as big-endian.
macro_rules! impl_for_integer {
    ($type: ident) => {
        impl WormDecode for $type {
            #[inline]
            fn decode_reader<R: Read>(reader: &mut R) -> Result<Self> {
                let mut buf = [0u8; size_of::<$type>()];
                reader
                    .read_exact(&mut buf)
                    .map_err(unexpected_eof_to_unexpected_length_of_input)?;
                let res = $type::from_be_bytes(buf.try_into().unwrap());
                Ok(res)
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

impl WormDecode for bool {
    #[inline]
    fn decode_reader<R: Read>(reader: &mut R) -> Result<Self> {
        let b: u8 = WormDecode::decode_reader(reader)?;
        if b == 0 {
            Ok(false)
        } else if b == 1 {
            Ok(true)
        } else {
            let msg = format!("Invalid bool representation: {}", b);

            Err(Error::new(ErrorKind::InvalidInput, msg))
        }
    }
}

impl<T, const N: usize> WormDecode for [T; N]
where
    T: WormDecode,
{
    #[inline]
    fn decode_reader<R: Read>(reader: &mut R) -> Result<Self> {
        struct ArrayDropGuard<T, const N: usize> {
            buffer: [core::mem::MaybeUninit<T>; N],
            init_count: usize,
        }
        impl<T, const N: usize> Drop for ArrayDropGuard<T, N> {
            fn drop(&mut self) {
                let init_range = &mut self.buffer[..self.init_count];
                // SAFETY: Elements up to self.init_count have been initialized. Assumes this value
                //         is only incremented in `fill_buffer`, which writes the element before
                //         increasing the init_count.
                unsafe {
                    core::ptr::drop_in_place(init_range as *mut _ as *mut [T]);
                };
            }
        }
        impl<T, const N: usize> ArrayDropGuard<T, N> {
            unsafe fn transmute_to_array(mut self) -> [T; N] {
                debug_assert_eq!(self.init_count, N);
                // Set init_count to 0 so that the values do not get dropped twice.
                self.init_count = 0;
                // SAFETY: This cast is required because `mem::transmute` does not work with
                //         const generics https://github.com/rust-lang/rust/issues/61956. This
                //         array is guaranteed to be initialized by this point.
                core::ptr::read(&self.buffer as *const _ as *const [T; N])
            }
            fn fill_buffer(&mut self, mut f: impl FnMut() -> Result<T>) -> Result<()> {
                // TODO: replace with `core::array::try_from_fn` when stabilized to avoid manually
                // dropping uninitialized values through the guard drop.
                for elem in self.buffer.iter_mut() {
                    elem.write(f()?);
                    self.init_count += 1;
                }
                Ok(())
            }
        }

        if let Some(arr) = T::array_from_reader(reader)? {
            Ok(arr)
        } else {
            let mut result = ArrayDropGuard {
                buffer: unsafe { core::mem::MaybeUninit::uninit().assume_init() },
                init_count: 0,
            };

            result.fill_buffer(|| T::decode_reader(reader))?;

            // SAFETY: The elements up to `i` have been initialized in `fill_buffer`.
            Ok(unsafe { result.transmute_to_array() })
        }
    }
}

impl WormDecode for Pubkey {
    fn decode_reader<R: Read>(reader: &mut R) -> Result<Self> {
        <[u8; 32]>::decode_reader(reader).map(Into::into)
    }
}
