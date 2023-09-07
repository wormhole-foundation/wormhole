//! Zero-copy account structs for the Core Bridge Program. Wherever possible, it is recommended to
//! use these structs instead of using Anchor's [Account](anchor_lang::prelude::Account) to
//! deserialize these accounts.

mod config;
pub use config::*;

mod message;
pub use message::*;

mod vaa;
pub use vaa::*;

use anchor_lang::prelude::{AccountInfo, Result};

pub trait LoadZeroCopy<'a>: Sized {
    fn load(acc_info: &'a AccountInfo) -> Result<Self>;
}
