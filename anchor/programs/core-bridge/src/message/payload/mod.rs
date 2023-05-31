mod decode;
mod encode;

pub use {decode::WormDecode, encode::WormEncode};

// pub trait WormPayload: WormDecode + WormEncode {}
// impl<T: WormDecode + WormEncode> WormPayload for T {}
